/*
Resource abstractions.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package backend

import (
	"database/sql"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var InvalidResource error = fmt.Errorf("Invalid resource\n")
var InvalidBody error = fmt.Errorf("Invalid body\n")
var InvalidMethod error = fmt.Errorf("Invalid method\n")

// get/set permission types.
const (
	get = 1 << iota
	set
	create
	delete
)

const (
	dbNameLen = 128
	dbDescLen = 512
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
	get(Encoder) error
	set(Decoder) error
	create(Decoder, func(string, interface{}) error) error
	delete() error
}

// Fake encoder to allow extracting the current state from a get call.
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


// resource provides a default implementation of all of the methods required
// to implement Resource.
type resource struct {}

func (r resource) Permissions() int {
	return get | set | create | delete
}

func (r resource) get(enc Encoder) error {
	return InvalidMethod
}

func (r resource) set(dec Decoder) error {
	return InvalidMethod
}

func (r resource) create(dec Decoder, success func(string, interface{}) error) error {
	return InvalidMethod
}

func (r resource) delete() error {
	return InvalidMethod
}

type login struct {
	Resource
	user string
	db   *sql.DB
}

// FIXME: Implement set as a way of changing passwords.
// FIXME: Implement delete as a way of deleting an account.
// FIXME: Figure out how to move the login creation from authenticateUser to
// create here.

func (l *login) get(enc Encoder) error {
	return nil // No-op - for checking login credentials.
}

type projectList struct {
	Resource
	user string
	permissions int
	db   *sql.DB
}

func (l *projectList) Permissions() int {
	return l.permissions
}

func (l *projectList) get(enc Encoder) error {
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

// create a new project.
func (l *projectList) create(dec Decoder, success func(string, interface{}) error) error {
	project := project{}
	err := dec.Decode(&project)
	if err != nil || ! project.valid() {
		return InvalidBody
	}
	project.Pid = uint(rand.Int())
	_, err = l.db.Exec("INSERT INTO projects VALUES (?, ?, ?, ?, ?, ?)",
		project.Pid, project.Name, project.Percentage, project.Description,
		false, 0)
	if err != nil {
		return err
	}

	// Add the user to the project.
	_, err = l.db.Exec("INSERT INTO owns VALUES (?, ?)", l.user, project.Pid)
	if err != nil {
		return err
	}
	return success(fmt.Sprintf("/projects/%d", project.Pid), project)
}

func NewProjectList(user string, db *sql.DB) (Resource, error) {
	p := projectList{resource{}, user, get, db}
	// Check if the user is a manager.
	is_manager := false
	err := db.QueryRow("SELECT is_manager FROM users WHERE name=?", user).Scan(&is_manager)
	if err != nil {
		return nil, err
	}
	if is_manager {
		p.permissions |= create
	}
	return &p, nil
}

type projectResource struct {
	Resource
	pid         uint
	permissions int
	db          *sql.DB
	user        string
}

type project struct {
	Pid uint
	Name string
	Percentage uint
	Description string
	Owns bool
}

// valid returns true if the given project looks like it should fit in the
// database with no errors.
func (p project) valid() bool {
	return (p.Percentage <= 100) &&
		(len(p.Name) < dbNameLen) && (len(p.Name) > 0) &&
		(len(p.Description) < dbDescLen) && (len(p.Description) > 0)
}

func (p *projectResource) Permissions() int {
	return p.permissions
}

func (p *projectResource) get(enc Encoder) error {
	name, percentage, description := "", 0, ""
	err := p.db.QueryRow("SELECT name, percentage, description FROM projects WHERE id=?", p.pid).
		Scan(&name, &percentage, &description)
	if err != nil {
		return err
	}
	project := project{p.pid, name, uint(percentage), description, p.permissions&set != 0}
	return enc.Encode(project)
}

// set the project state on the server.
// We override any existing state as I have not implemented any kind
// of synchronisation.
// FIXME: Add synchronisation.
func (p *projectResource) set(dec Decoder) error {
	project := project{}
	err := dec.Decode(&project)
	if err != nil || ! project.valid() || project.Pid != p.pid {
		return InvalidBody
	}
	_, err = p.db.Exec("UPDATE projects SET name=?, percentage=?, description=? WHERE id=?",
		project.Name, project.Percentage, project.Description, p.pid)
	return err
}

// delete the given project from the current user.
// This should remove the current user from the project.
// If there are no managers left for the given project, delete it, any
// deliverables, and any viewing relations involving it.
func (p *projectResource) delete() error {
	var err error = nil
	if p.permissions&set == 0 {
		// Not an owner.
		_, err = p.db.Exec("DELETE FROM views WHERE name=? and pid=?",
			p.user, p.pid)
	} else {
		// Project owner.
		_, err = p.db.Exec("DELETE FROM owns WHERE name=? and pid=?",
			p.user, p.pid)
		if err != nil {
			return err
		}
		// Check for other managers.
		dbpid := 0
		err = p.db.QueryRow("SELECT pid FROM owns WHERE name=? and pid=?",
			p.user, p.pid).Scan(&dbpid)
		if err == sql.ErrNoRows {
			// Remove any viewers.
			_, err = p.db.Exec("DELETE FROM views WHERE pid=?", p.pid)
			if err != nil {
				return err
			}
			// Remove any deliverables.
			_, err = p.db.Exec("DELETE FROM deliverables WHERE pid=?", p.pid)
			if err != nil {
				return err
			}
			// Remove the project.
			_, err = p.db.Exec("DELETE FROM projects WHERE id=?", p.pid)
		}
	}
	return err
}

func NewProject(user string, pid uint, db *sql.DB) (Resource, error) {
	p := projectResource{resource{}, pid, 0, db, user}

	// Find the user.
	dbpid := 0
	for _, table := range []string{"views", "owns"} {
		err := db.QueryRow(fmt.Sprintf("SELECT pid FROM %s WHERE name=? and pid=?", table), user, pid).Scan(&dbpid)
		if err == nil {
			if table == "owns" {
				p.permissions |= get | set | delete
			} else {
				p.permissions |= get | delete
			}
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}

	return &p, nil
}

type flagResource struct {
	Resource
	pid     uint
	project Resource
	db      *sql.DB
}

type flag struct {
	Version uint
	Value   bool
}

func (f *flagResource) Permissions() int {
	// Everyone can read and write to the flag.
	if get&f.project.Permissions() != 0 {
		return get | set
	}
	return 0
}

func (f *flagResource) get(enc Encoder) error {
	flag := flag{0, false}
	err := f.db.QueryRow("SELECT flag, flag_version FROM projects WHERE id=?", f.pid).Scan(&(flag.Value), &(flag.Version))
	if err != nil {
		return err
	}
	return enc.Encode(flag)
}

func (f *flagResource) set(dec Decoder) error {
	// Decode the uploaded flag.
	update := flag{0, false}
	err := dec.Decode(&update)
	if err != nil {
		return InvalidBody
	}

	// get the saved flag.
	cur := flag{0, false}
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

func NewFlag(user string, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &flagResource{resource{}, pid, proj, db}, err
}

type clients struct {
	Resource
	pid     uint
	project Resource
	db      *sql.DB
}

func (c *clients) Permissions() int {
	if c.project.Permissions()&set != 0 {
		return get | set
	}
	return 0
}

func (c *clients) get(enc Encoder) error {
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

func (c *clients) set(dec Decoder) error {
	// TODO: Implement syncronisation.

	// Populate the list of users in the database.
	old := map[string]bool{} // user->removed
	enc := MapEncoder{old}
	err := c.get(&enc)
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
			// We first check that the user actually exists.
			dbuser := ""
			err = c.db.QueryRow("SELECT name FROM users WHERE name=?", user).
				Scan(&dbuser)
			if err == sql.ErrNoRows {
				return InvalidBody
			} else if err != nil {
				return err
			}
			// Now add the user.
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

func NewClients(user string, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &clients{resource{}, pid, proj, db}, err
}

type deliverableList struct {
	Resource
	pid     uint
	project Resource
	db      *sql.DB
}

func (l *deliverableList) Permissions() int {
	if set&l.project.Permissions() != 0 {
		return get | create
	} else if get&l.project.Permissions() != 0 {
		return get
	}
	return 0
}

func (l *deliverableList) get(enc Encoder) error {
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

// create for deliverableList creates a new deliverable.
func (l *deliverableList) create(dec Decoder, success func(string, interface{}) error) error {
	v := deliverable{}
	err := dec.Decode(&v)
	if err != nil || !v.valid() {
		return InvalidBody
	}
	v.Id = uint(rand.Int())
	_, err = l.db.Exec("INSERT INTO deliverables VALUES (?, ?, ?, ?, ?, ?)",
		v.Id, l.pid, v.Name, v.Due, v.Percentage, v.Description)
	if err != nil {
		return err
	}
	return success(fmt.Sprintf("/projects/%d/deliverables/%d", l.pid, v.Id), v)
}

func NewDeliverableList(user string, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &deliverableList{resource{}, pid, proj, db}, err
}

type deliverableResource struct {
	Resource
	id      uint
	pid     uint
	project Resource
	db      *sql.DB
}

type deliverable struct {
	Id          uint
	Name        string
	Due         string
	Percentage  uint
	Description string
}

// valid of deliverables returns true if the value will fit in the database and
// is valid.
// FIXME: Validate the Due value.
func (d deliverable) valid() bool {
	return (d.Percentage <= 100) &&
		(len(d.Name) < dbNameLen) && (len(d.Name) > 0) &&
		(len(d.Description) < dbDescLen) && (len(d.Description) > 0)
}

func (d *deliverableResource) Permissions() int {
	if set&d.project.Permissions() != 0 {
		return get | set | create | delete
	}
	return get&d.project.Permissions()
}

func (d *deliverableResource) get(enc Encoder) error {
	v := deliverable{}
	err := d.db.QueryRow("SELECT name, due, percentage, description FROM deliverables WHERE id=? and pid=?", d.id, d.pid).
		Scan(&v.Name, &v.Due, &v.Percentage, &v.Description)
	if err != nil {
		return err
	}
	return enc.Encode(v)
}

func (d *deliverableResource) set(dec Decoder) error {
	v := deliverable{}
	err := dec.Decode(&v)
	if err != nil || !v.valid() {
		return InvalidBody
	}
	_, err = d.db.Exec("UPDATE deliverables SET name=?, due=?, percentage=?, description=? WHERE id=? and pid=?",
		v.Name, v.Due, v.Percentage, v.Description, d.id, d.pid)
	return err
}

func (d *deliverableResource) delete() error {
	_, err := d.db.Exec("DELETE FROM deliverables WHERE id=? and pid=?",
		d.id, d.pid)
	return err
}

func NewDeliverable(user string, id uint, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	if err != nil {
		return nil, err
	}

	// Check that the deliverable actually exists.
	dbpid := 0
	err = db.QueryRow("SELECT pid FROM deliverables WHERE id=? and pid=?", id, pid).Scan(&dbpid)
	if err == sql.ErrNoRows {
		return nil, InvalidResource
	} else if err != nil {
		return nil, err
	}
	return &deliverableResource{resource{}, id, pid, proj, db}, nil
}

// FromURI returns the resource corresponding to the given URI.
func FromURI(user, uri string, db *sql.DB) (Resource, error) {
	// Match the path to the regular expressions.
	if loginRe.MatchString(uri) {
		return &login{resource{}, user, db}, nil
	} else if projectListRe.MatchString(uri) {
		return NewProjectList(user, db)
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

// Seed the PRNG.
// This *must* be called before using FromURI.
func Seed() {
	rand.Seed(time.Now().Unix())
}

// vim: sw=4 ts=4 noexpandtab
