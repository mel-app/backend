/*
MEL app backend.



Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"fmt"
	"html"
	"log"
	"net/http"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

// Authenticate the given HTTP request.
func authenticate(fail func(int), request *http.Request, db *sql.DB) (string, bool) {
	name, password, ok := request.BasicAuth()
	if !ok {
		fail(http.StatusUnauthorized)
		return name, ok
	}

	// dbname and dbpassword are empty values to pass to Scan; we never use them
	// elsewhere.
	dbname := ""
	// FIXME: This is not "best-practice".
	//	We should salt the password (using a locally stored value), and maybe
	//	use encrypt(name+password) to avoid duplicated passwords being obvious?
	err := db.QueryRow("SELECT name FROM users WHERE name=? and password=?", name, password).Scan(&dbname)
	if err == sql.ErrNoRows {
		log.Printf("Failed to authenticate %q\n", name) // TODO: Remove?
		fail(http.StatusForbidden)
	} else if err != nil {
		log.Printf("Error authenticating user: %q\n", err)
		fail(http.StatusInternalServerError)
	}

	return name, err == nil
}

// Handle a single HTTP request.
func handle(writer http.ResponseWriter, request *http.Request) {
	log.Printf("Handling request for %q\n", html.EscapeString(request.URL.Path))

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
	name, ok := authenticate(fail, request, db)
	if !ok {
		return
	}
	fmt.Fprintf(writer, "%q: authenticated as %s\n",
		html.EscapeString(request.URL.Path), name)

	// Parse the URL and return the corresponding value.
}

func main() {
	fmt.Printf("Starting server on :8080...\n")
	http.ListenAndServe(":8080", http.HandlerFunc(handle))
}

// vim: sw=4 ts=4 noexpandtab
