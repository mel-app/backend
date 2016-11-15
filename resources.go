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
	_ "strconv"

	_ "github.com/mattn/go-sqlite3"
)

var InvalidResource error = fmt.Errorf("Invalid resource\n")

// Read/Write permission types.
const (
	Read = 1 << iota
	Write
)

// Interface for the various resource types.
type Resource interface {
	Permissions() int
}

// Regular expressions for the various resources.
var (
	projectListRe = regexp.MustCompile("/projects")
	projectRe = regexp.MustCompile("/projects/(\\d+)")
	flagRe = regexp.MustCompile("/projects/(\\d+)/flag")
	deliverableRe = regexp.MustCompile("/projects/(\\d+)/deliverables")
)

type projectList struct {
	user string
}


func (_ *projectList) Permissions() int {
	return Read | Write
}


// FromURI returns the resource corresponding to the given URI.
func FromURI(user, uri string, db *sql.DB) (Resource, error) {
	// Match the path to the regular expressions.
	if projectListRe.MatchString(uri) {
		return &projectList{user}, nil
	} else if projectRe.MatchString(uri) {
		return nil, fmt.Errorf("Haven't implemented this yet...")
	} else {
		return nil, InvalidResource
	}
}

// vim: sw=4 ts=4 noexpandtab
