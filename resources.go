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

// Read/Write permission types.
const (
	Read = 1 << iota
	Write
)

// Interface abstracting encoders.
type Encoder interface {
	Encode(interface{}) error
}

// Interface for the various resource types.
type Resource interface {
	Permissions() int    // Return a combination of Read and Write.
	Write(Encoder) error // Writes the resource to the given encoder.
}

// Regular expressions for the various resources.
var (
	projectListRe = regexp.MustCompile(`\A/projects/?\z`)
	projectRe     = regexp.MustCompile(`\A/projects/(\d+)/?\z`)
	flagRe        = regexp.MustCompile(`\A/projects/(\d+)/flag\z`)
	deliverableRe = regexp.MustCompile(`\A/projects/(\d+)/deliverables\z`)
)

type projectList struct {
	user string
	db   *sql.DB
}

func (_ *projectList) Permissions() int {
	return Read | Write
}

func (l *projectList) Write(enc Encoder) error {
	rows, err := l.db.Query("SELECT pid FROM views WHERE name=?", l.user)
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

func (p *project) Write(enc Encoder) error {
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

func NewProject(user string, pid uint, db *sql.DB) (Resource, error) {
	p := project{pid, 0, db}
	dbpid := 0
	err := db.QueryRow("SELECT pid FROM views WHERE name=? and pid=?", user, pid).Scan(&dbpid)
	if err == nil {
		p.permissions |= Read
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
	if Read&f.project.Permissions() != 0 {
		return Read | Write
	}
	return 0
}

func (f *flag) Write(enc Encoder) error {
	flag := false
	err := f.db.QueryRow("SELECT flag FROM projects WHERE id=?", f.pid).Scan(&flag)
	if err != nil {
		return err
	}
	return enc.Encode(flag)
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
