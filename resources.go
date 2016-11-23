/*
Resource abstractions.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

var InvalidResource error = fmt.Errorf("Invalid resource\n")
var InvalidBody error = fmt.Errorf("Invalid body\n")
var InvalidMethod error = fmt.Errorf("Invalid method\n")

// Get/Set permission types.
const (
	Get = 1 << iota
	Set
	Create
)

// Interface abstracting encoders.
type Encoder interface {
	Encode(interface{}) error
}
type Decoder interface {
	Decode(interface{}) error
	More() bool
}

// Interface for the various resource types.
type Resource interface {
	Permissions() int
	Get(Encoder) error
	Set(Decoder) error
	Create(Decoder) error
}

// Fake encoder to allow extracting the current state from a Get call.
type MapEncoder struct {
	current map[string]bool
}

func (m *MapEncoder) Encode(item interface{}) error {
	// FIXME: This is pretty ugly and inflexible. Perhaps use reflection
	//  instead?
	m.current[fmt.Sprintf("%v", item)] = true
	return nil
}

// Regular expressions for the various resources.
var (
	loginRe           = regexp.MustCompile(`\A/login\z`)
	projectListRe     = regexp.MustCompile(`\A/projects\z`)
	projectRe         = regexp.MustCompile(`\A/projects/(\d+)\z`)
	flagRe            = regexp.MustCompile(`\A/projects/(\d+)/flag\z`)
	clientsRe         = regexp.MustCompile(`\A/projects/(\d+)/clients\z`)
	deliverableListRe = regexp.MustCompile(`\A/projects/(\d+)/deliverables\z`)
	deliverableRe     = regexp.MustCompile(`\A/projects/(\d+)/deliverables/(\d+)\z`)
)

type projectList struct {
	user string
	db   *sql.DB
}

func (_ *projectList) Permissions() int {
	return Get
}

func (l *projectList) Get(enc Encoder) error {
	for _, table := range []string{"views", "owns"} {
		rows, err := l.db.Query(fmt.Sprintf("SELECT pid FROM %s WHERE name=?", table), l.user)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			id := -1
			err = rows.Scan(&id)
			if err != nil {
				return err
			}
			err = enc.Encode(id)
			if err != nil {
				return err
			}
		}
		if rows.Err() != nil {
			return rows.Err()
		}
	}
	return nil
}

func (l *projectList) Set(dec Decoder) error {
	return InvalidMethod
}

func (l *projectList) Create(dec Decoder) error {
	return InvalidMethod
}

type login struct {
	user string
	db   *sql.DB
}

func (l *login) Permissions() int {
	return Get | Set | Create
}

func (l *login) Get(enc Encoder) error {
	return nil // No-op - for checking login credentials.
}

func (l *login) Set(dec Decoder) error {
	// FIXME: Implement this as a way of changing user passwords.
	return InvalidMethod
}

func (l *login) Create(dec Decoder) error {
	// FIXME: Figure out how to move the login creation from authenticateUser.
	return nil // Implemented in backend.go as a special case.
}

type projectResource struct {
	pid         uint
	permissions int
	db          *sql.DB
	user        string
}

func (p *projectResource) Permissions() int {
	return p.permissions
}

func (p *projectResource) Get(enc Encoder) error {
	name, percentage, description := "", "", ""
	err := p.db.QueryRow("SELECT name, percentage, description FROM projects WHERE id=?", p.pid).Scan(&name, &percentage, &description)
	if err != nil {
		return err
	}
	err = enc.Encode(name)
	if err != nil {
		return err
	}
	err = enc.Encode(percentage)
	if err != nil {
		return err
	}
	return enc.Encode(description)
}

// Set the project state on the server.
// We override any existing state as I have not implemented any kind
// of synchronisation.
// FIXME: Add synchronisation.
func (p *projectResource) Set(dec Decoder) error {
	name, percentage, description := "", "", ""
	err := dec.Decode(&name)
	if err != nil {
		return InvalidBody
	}
	err = dec.Decode(&percentage)
	if err != nil {
		return InvalidBody
	}
	err = dec.Decode(&description)
	if err != nil {
		return InvalidBody
	}
	_, err = p.db.Exec("UPDATE projects SET name=?, percentage=?, description=? WHERE id=?", name, percentage, description, p.pid)
	return err
}

// Create a new project on the server, and assign the user as the owner.
func (p *projectResource) Create(dec Decoder) error {
	// Begin by generating an unused ID for the project.
	// TODO: This is ugly and probably prone to race conditions.
	// (no locking between requests).
	id := -1
	var err error = nil
	for err != sql.ErrNoRows {
		id += 1
		err = p.db.QueryRow("SELECT id FROM projects WHERE id=?", id).Scan(&id)
	}

	// Now create the project.
	name, percentage, description := "", "0", ""
	err = dec.Decode(&name)
	if err != nil {
		return InvalidBody
	}
	err = dec.Decode(&percentage)
	if err != nil {
		return InvalidBody
	}
	err = dec.Decode(&description)
	if err != nil {
		return InvalidBody
	}
	_, err = p.db.Exec("INSERT INTO projects VALUES (?, ?, ?, ?, ?)", id, name, percentage, description, false)
	if err != nil {
		return err
	}

	// Add the user to the project.
	_, err = p.db.Exec("INSERT INTO owns VALUES (?, ?)", p.user, id)
	return err
}

func NewProject(user string, pid uint, db *sql.DB) (Resource, error) {
	p := projectResource{pid, 0, db, user}

	// Find the user.
	dbpid := 0
	for _, table := range []string{"views", "owns"} {
		err := db.QueryRow(fmt.Sprintf("SELECT pid FROM %s WHERE name=? and pid=?", table), user, pid).Scan(&dbpid)
		if err == nil {
			if table == "owns" {
				p.permissions |= Get | Set
			} else {
				p.permissions |= Get
			}
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}

	// Check if the user is a manager.
	is_manager := false
	err := db.QueryRow("SELECT is_manager FROM users WHERE name=?", user).Scan(&is_manager)
	if err == nil {
		if is_manager {
			p.permissions |= Create
		}
	} else {
		return nil, err
	}
	return &p, nil
}

type flag struct {
	pid     uint
	project Resource
	db      *sql.DB
}

type versionedFlag struct {
	Version uint
	Value   bool
}

func (f *flag) Permissions() int {
	// Everyone can read and write to the flag.
	if Get&f.project.Permissions() != 0 {
		return Get | Set
	}
	return 0
}

func (f *flag) Get(enc Encoder) error {
	flag := versionedFlag{0, false}
	err := f.db.QueryRow("SELECT flag, flag_version FROM projects WHERE id=?", f.pid).Scan(&(flag.Value), &(flag.Version))
	if err != nil {
		return err
	}
	return enc.Encode(flag)
}

func (f *flag) Set(dec Decoder) error {
	// Decode the uploaded flag.
	update := versionedFlag{0, false}
	err := dec.Decode(&update)
	if err != nil {
		return InvalidBody
	}

	// Get the saved flag.
	cur := versionedFlag{0, false}
	err = f.db.QueryRow("SELECT flag, flag_version FROM projects WHERE id=?", f.pid).Scan(&(cur.Value), &(cur.Version))
	if err != nil {
		return err
	}

	// Reject invalid versions.
	if update.Version > cur.Version {
		return InvalidBody
	}

	// Compare and sync.
	// If the version from the client is equal to the version on the server,
	// use the value from the client and increment the server version.
	// Otherwise, just use the server version.
	if update.Version == cur.Version && update.Value != cur.Value {
		_, err = f.db.Exec("UPDATE projects SET flag=?, flag_version=? WHERE id=?",
			update.Value, update.Version+1, f.pid)
		return err
	}
	return nil
}

func (f *flag) Create(dec Decoder) error {
	return InvalidMethod // Can't create a flag.
}

func NewFlag(user string, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &flag{pid, proj, db}, err
}

type clients struct {
	pid     uint
	project Resource
	db      *sql.DB
}

func (c *clients) Permissions() int {
	if c.project.Permissions()&Set != 0 {
		return Get | Set
	}
	return 0
}

func (c *clients) Get(enc Encoder) error {
	rows, err := c.db.Query("SELECT name FROM views WHERE pid=?", c.pid)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		id := ""
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		err = enc.Encode(id)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func (c *clients) Set(dec Decoder) error {
	// TODO: Implement syncronisation.
	// FIXME: We don't validate the user names.

	// Populate the list of users in the database.
	old := map[string]bool{} // user->removed
	enc := MapEncoder{old}
	err := c.Get(&enc)
	if err != nil {
		return err
	}

	// Find added clients and update the list.
	for dec.More() {
		user := ""
		err = dec.Decode(&user)
		if err != nil {
			return InvalidBody
		}
		if _, ok := old[user]; !ok {
			// New user; add it to the database.
			_, err = c.db.Exec("INSERT INTO views VALUES (?, ?)", user, c.pid)
			if err != nil {
				return err
			}
		} else {
			// Mark the user as 'found'
			old[user] = false
		}
	}

	// Find old clients and remove them.
	for key, removed := range old {
		if removed {
			_, err = c.db.Exec("DELETE FROM views WHERE name=? and pid=?", key, c.pid)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *clients) Create(dec Decoder) error {
	return InvalidMethod // Can't create a clients resource.
}

func NewClients(user string, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &clients{pid, proj, db}, err
}

type deliverableList struct {
	pid     uint
	project Resource
	db      *sql.DB
}

func (l *deliverableList) Permissions() int {
	if Set&l.project.Permissions() != 0 {
		return Get | Set
	} else if Get&l.project.Permissions() != 0 {
		return Get
	}
	return 0
}

func (l *deliverableList) Get(enc Encoder) error {
	rows, err := l.db.Query("SELECT id FROM deliverables WHERE pid=?", l.pid)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		id := -1
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		err = enc.Encode(id)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func (l *deliverableList) Set(dec Decoder) error {
	return InvalidMethod
}

func (l *deliverableList) Create(dec Decoder) error {
	return InvalidMethod // Can't create a deliverable list.
}

func NewDeliverableList(user string, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &deliverableList{pid, proj, db}, err
}

type deliverable struct {
	id      uint
	pid     uint
	project Resource
	db      *sql.DB
}

type deliverableValue struct {
	name        string
	due         string
	percentage  uint
	description string
}

func (d *deliverable) Permissions() int {
	if Set&d.project.Permissions() != 0 {
		return Get | Set | Create
	} else if Get&d.project.Permissions() != 0 {
		return Get
	}
	return 0
}

func (d *deliverable) Get(enc Encoder) error {
	v := deliverableValue{}
	err := d.db.QueryRow("SELECT name, due, percentage, description FROM deliverables WHERE id=? and pid=?", d.id, d.pid).
		Scan(&v.name, &v.due, &v.percentage, &v.description)
	// TODO: We don't validate the resource name while creating it so this
	//		can crash dramatically...
	if err != nil {
		return err
	}
	return enc.Encode(v)
}

func (d *deliverable) Set(dec Decoder) error {
	v := deliverableValue{}
	err := dec.Decode(&v)
	if err != nil {
		return InvalidBody
	}
	_, err = d.db.Exec("UPDATE deliverables SET name=?, due=?, percentage=?, description=? WHERE id=? and pid=?",
		v.name, v.due, v.percentage, v.description, d.id, d.pid)
	return err
}

func (d *deliverable) Create(dec Decoder) error {
	// Begin by generating an unused ID for the deliverable.
	// TODO: This is ugly and probably prone to race conditions.
	// (no locking between requests).
	// FIXME: This is largely duplicated from the project creation code.
	id := -1
	var err error = nil
	for err != sql.ErrNoRows {
		id += 1
		err = d.db.QueryRow("SELECT id FROM deliverables WHERE pid=? and id=?", d.pid, id).Scan(&id)
	}

	// Now create the project.
	v := deliverableValue{}
	err = dec.Decode(&v)
	if err != nil {
		return InvalidBody
	}
	_, err = d.db.Exec("INSERT INTO deliverables VALUES (?, ?, ?, ?, ?)",
		id, d.pid, v.name, v.due, v.percentage, v.description)
	// TODO: This may not be an "internal server error" - check first.
	return err
}

func NewDeliverable(user string, id uint, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &deliverable{id, pid, proj, db}, err
}

// FromURI returns the resource corresponding to the given URI.
func FromURI(user, uri string, db *sql.DB) (Resource, error) {
	// Match the path to the regular expressions.
	if projectListRe.MatchString(uri) {
		return &projectList{user, db}, nil
	} else if loginRe.MatchString(uri) {
		return &login{user, db}, nil
	} else if projectRe.MatchString(uri) {
		pid, err := strconv.Atoi(projectRe.FindStringSubmatch(uri)[1])
		if err != nil {
			return nil, err
		}
		return NewProject(user, uint(pid), db)
	} else if flagRe.MatchString(uri) {
		pid, err := strconv.Atoi(flagRe.FindStringSubmatch(uri)[1])
		if err != nil {
			return nil, err
		}
		return NewFlag(user, uint(pid), db)
	} else if clientsRe.MatchString(uri) {
		pid, err := strconv.Atoi(clientsRe.FindStringSubmatch(uri)[1])
		if err != nil {
			return nil, err
		}
		return NewClients(user, uint(pid), db)
	} else if deliverableListRe.MatchString(uri) {
		pid, err := strconv.Atoi(deliverableListRe.FindStringSubmatch(uri)[1])
		if err != nil {
			return nil, err
		}
		return NewDeliverableList(user, uint(pid), db)
	} else if deliverableRe.MatchString(uri) {
		pid, err := strconv.Atoi(deliverableRe.FindStringSubmatch(uri)[1])
		if err != nil {
			return nil, err
		}
		id, err := strconv.Atoi(deliverableRe.FindStringSubmatch(uri)[2])
		if err != nil {
			return nil, err
		}
		return NewDeliverable(user, uint(id), uint(pid), db)
	} else {
		return nil, InvalidResource
	}
}

// vim: sw=4 ts=4 noexpandtab
