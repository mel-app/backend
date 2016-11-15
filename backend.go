/*
MEL app backend.



Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"strconv"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

// internalError ends the request and logs an internal error.
func internalError(fail func(int), err error) {
	fail(http.StatusInternalServerError)
	log.Printf("%q\n", err)
}

// Authenticate the given HTTP request.
func authenticate(fail func(int), request *http.Request, db *sql.DB) (string, bool) {
	user, password, ok := request.BasicAuth()
	if !ok {
		fail(http.StatusUnauthorized)
		return user, ok
	}

	// dbname and dbpassword are empty values to pass to Scan; we never use them
	// elsewhere.
	dbname := ""
	// FIXME: This is not "best-practice".
	//	We should salt the password (using a locally stored value), and maybe
	//	use encrypt(name+password) to avoid duplicated passwords being obvious?
	err := db.QueryRow("SELECT name FROM users WHERE name=? and password=?", user, password).Scan(&dbname)
	if err == sql.ErrNoRows {
		fail(http.StatusForbidden)
	} else if err != nil {
		internalError(fail, err)
	}

	return user, err == nil
}

// ListProjects responds with the list of projects for the given user.
func ListProjects(fail func(int), encoder *json.Encoder, user string, db *sql.DB) {
	// TODO: This should also return projects which this user owns.
	//	Implement that as a view in the database?
	rows, err := db.Query("SELECT id FROM views WHERE name=?", user)
	if err != nil {
		internalError(fail, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		id := -1
		err = rows.Scan(&id)
		if err != nil {
			internalError(fail, err)
			return
		}
		err = encoder.Encode(id)
		if err != nil {
			internalError(fail, err)
			return
		}
	}
	if err != nil {
		internalError(fail, err)
		return
	}
}

// Flag responds with the current state of the flag.
func Flag(fail func(int), encoder *json.Encoder, pid int, user string, db *sql.DB) {
	// TODO: We need to authenticate the user here.
	flag := false
	err := db.QueryRow("SELECT flag FROM project WHERE id=?", pid).Scan(&flag)
	if err != nil {
		internalError(fail, err)
		return
	}
	encoder.Encode(flag)
}

// Project responds with the details of the given project.
func Project(fail func(int), encoder *json.Encoder, pid int, user string, db *sql.DB) {
	// TODO: We need to authenticate the user here.
	name, percentage, description := "", "", ""
	err := db.QueryRow("SELECT name, percentage, description FROM project WHERE id=?", pid).Scan(&name, &percentage, &description)
	if err != nil {
		internalError(fail, err)
		return
	}
	encoder.Encode(name)
	encoder.Encode(percentage)
	encoder.Encode(description)
}

// Handle a single HTTP request.
func handle(writer http.ResponseWriter, request *http.Request) {
	// Wrapper for failing functions.
	fail := func(status int) { http.Error(writer, http.StatusText(status), status) }

	// Open the database.
	// FIXME: I'm using sqlite3 here which only seems to report errors when
	//	actually executing a query; I'll need to test this on other systems as well.
	db, err := sql.Open("sqlite3", "test.db") // TODO: Should be the actual db, ...
	if err != nil {
		log.Printf("Error opening DB: %q\n", err)
		fail(http.StatusInternalServerError)
	}

	// Authenticate.
	user, ok := authenticate(fail, request, db)
	if !ok {
		return
	}

	// Parse the URL and return the corresponding value.
	// TODO: This assumes GET requests...
	enc := json.NewEncoder(writer)
	enc.SetEscapeHTML(true)
	paths := strings.Split(strings.TrimPrefix(request.URL.Path, "/"), "/")

	// FIXME: Match using regular expressions instead?
	if len(paths) < 1 || paths[0] != "projects" {
		http.NotFound(writer, request)
	} else if len(paths) == 1 {
		ListProjects(fail, enc, user, db)
	} else {
		// Grab the project ID from the URL.
		pid, err := strconv.Atoi(paths[1])
		if err != nil {
			http.NotFound(writer, request)
			return
		}

		if len(paths) == 2 {
			Project(fail, enc, pid, user, db)
		} else if len(paths) == 3 && paths[2] == "flag" {
			Flag(fail, enc, pid, user, db)
		} else {
			http.NotFound(writer, request)
		}
	}
}

func main() {
	fmt.Printf("Starting server on :8080...\n")
	http.ListenAndServe(":8080", http.HandlerFunc(handle))
}

// vim: sw=4 ts=4 noexpandtab
