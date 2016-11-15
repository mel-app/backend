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
	"strconv"
	"strings"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/scrypt"
)

// internalError ends the request and logs an internal error.
func internalError(fail func(int), err error) {
	fail(http.StatusInternalServerError)
	log.Printf("%q\n", err)
}

// authenticateUser checks that the user and password in the given HTTP request.
func authenticateUser(fail func(int), request *http.Request, db *sql.DB) (user string, ok bool) {
	// Get the user name and password.
	user, password, ok := request.BasicAuth()
	if !ok {
		fail(http.StatusUnauthorized)
		return user, false
	}

	// Retrieve the salt and database password.
	salt := []byte("")
	dbpassword := []byte("")
	err := db.QueryRow("SELECT salt, password FROM users WHERE name=?", user).Scan(&salt, &dbpassword)
	if err == sql.ErrNoRows {
		fail(http.StatusForbidden)
		return user, false
	} else if err != nil {
		internalError(fail, err)
		return user, false
	}

	// Check the password. We salt and encrypt it to avoid potential security
	// issues if the db is stolen.
	// This appears to be reasonably close to "best practice", but the 1<<16
	// value probably should be checked for sanity.
	// FIXME: We don't store the 1<<16 value in the db, but it should be
	// increased as compute power grows. Doing so is complicated since some way
	// of migrating users from the old value would also need to be implemented.
	key, err := scrypt.Key([]byte(password), salt, 1<<16, 8, 1, 256)
	if err != nil {
		internalError(fail, err)
		return user, false
	}
	return user, string(key) == string(dbpassword)
}

// authenticateRequest checks that the given user has permission to complete
// the request.
func authenticateRequest(fail func(int), request *http.Request, db *sql.DB) ok bool {
	// TODO: Implement checking here.
	return true
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
	user, ok := authenticateUser(fail, request, db)
	if !ok || !authenticateRequest(fail, request, db) {
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
