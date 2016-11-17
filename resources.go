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

// Regular expressions for the various resources.
var (
	projectListRe = regexp.MustCompile(`\A/projects\z`)
	projectRe     = regexp.MustCompile(`\A/projects/(\d+)\z`)
	flagRe        = regexp.MustCompile(`\A/projects/(\d+)/flag\z`)
	deliverableRe = regexp.MustCompile(`\A/projects/(\d+)/deliverables\z`)
)

type projectList struct {
	user string
	db   *sql.DB
}

func (_ *projectList) Permissions() int {
	return Get | Set
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

// Set for a projectList allows a login to unsubscribe themselves from projects.
func (l *projectList) Set(dec Decoder) error {
	// TODO: We don't implement this as it is nontrivial...
	return nil
}

func (l *projectList) Create(dec Decoder) error {
	// This is implemented in the authentication code.
	return nil
}

// removeUser removes the given user from the given project.
// It also garbage-collects the project by decrementing and checking the
// viewing counter.
// FIXME: Implement the garbage collection...
func removeUser(user string, pid uint, db *sql.DB) error {
	_, err := db.Exec("DELETE FROM owns WHERE name=? and pid=?", user, pid)
	if err != nil { return err }
	_, err = db.Exec("DELETE FROM views WHERE name=? and pid=?", user, pid)
	return err
}

type project struct {
	pid         uint
	permissions int
	db          *sql.DB
}

func (p *project) Permissions() int {
	return p.permissions
}

func (p *project) Get(enc Encoder) error {
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
func (p *project) Set(dec Decoder) error {
	name, percentage, description := "", "", ""
	err := dec.Decode(&name)
	if err != nil {
		return err
	}
	err = dec.Decode(&percentage)
	if err != nil {
		return err
	}
	err = dec.Decode(&description)
	if err != nil {
		return err
	}
	_, err = p.db.Exec("UPDATE projects SET name=?, percentage=?, description=? WHERE pid=?", name, percentage, description, p.pid)
	return err
}

func (p *project) Create(dec Decoder) error {
	return nil
}

func NewProject(user string, pid uint, db *sql.DB) (Resource, error) {
	p := project{pid, 0, db}
	dbpid := 0
	for _, table := range []string{"views", "owns"} {
		err := db.QueryRow(fmt.Sprintf("SELECT pid FROM %s WHERE name=? and pid=?", table), user, pid).Scan(&dbpid)
		if err == nil {
			p.permissions |= Get
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}

	is_manager := false
	err := db.QueryRow("SELECT is_manager FROM users WHERE name=?", user).Scan(&is_manager)
	if err == nil {
		p.permissions |= Create
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	return &p, nil
}

type flag struct {
	pid     uint
	project Resource
	db      *sql.DB
}

func (f *flag) Permissions() int {
	// Everyone can read and write to the flag.
	if Get&f.project.Permissions() != 0 {
		return Get | Set
	}
	return 0
}

func (f *flag) Get(enc Encoder) error {
	flag := false
	err := f.db.QueryRow("SELECT flag FROM projects WHERE id=?", f.pid).Scan(&flag)
	if err != nil {
		return err
	}
	return enc.Encode(flag)
}

func (f *flag) Set(dec Decoder) error {
	return nil
}

func (f *flag) Create(dec Decoder) error {
	return nil
}

func NewFlag(user string, pid uint, db *sql.DB) (Resource, error) {
	proj, err := NewProject(user, pid, db)
	return &flag{pid, proj, db}, err
}

// FromURI returns the resource corresponding to the given URI.
func FromURI(user, uri string, db *sql.DB) (Resource, error) {
	// Match the path to the regular expressions.
	if projectListRe.MatchString(uri) {
		return &projectList{user, db}, nil
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
	} else {
		return nil, InvalidResource
	}
}

// vim: sw=4 ts=4 noexpandtab
