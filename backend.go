/*
MEL app backend.



Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/scrypt"
)

const passwordSize = 256

// internalError ends the request and logs an internal error.
func internalError(fail func(int), err error) {
	fail(http.StatusInternalServerError)
	log.Printf("%q\n", err)
}

// encryptPassword salts and encrypts the given password.
func encryptPassword(password string, salt []byte) ([]byte, error) {
	// We salt and encrypt the password to avoid potential security issues if
	// the db is stolen.
	// This appears to be reasonably close to "best practice", but the 1<<16
	// value probably should be checked for sanity.
	// FIXME: We don't store the 1<<16 value in the db, but it should be
	// increased as compute power grows. Doing so is complicated since some way
	// of migrating users from the old value would also need to be implemented.
	return scrypt.Key([]byte(password), salt, 1<<16, 8, 1, passwordSize)
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
	salt := make([]byte, passwordSize)
	dbpassword := []byte("")
	err := db.QueryRow("SELECT salt, password FROM users WHERE name=?", user).Scan(&salt, &dbpassword)
	if err == sql.ErrNoRows && request.URL.Path == "/projects" && request.Method == http.MethodPut {
		// FIXME: Special case creating a new user.
		_, err = rand.Read(salt)
		if err != nil {
			internalError(fail, err)
			return user, false
		}
		key, err := encryptPassword(password, salt)
		if err != nil {
			internalError(fail, err)
			return user, false
		}
		_, err = db.Exec("INSERT INTO users VALUES (?, ?, ?)", user, salt, key)
		if err != nil {
			internalError(fail, err)
			return user, false
		}
		return user, true
	} else if err == sql.ErrNoRows {
		fail(http.StatusForbidden)
		return user, false
	} else if err != nil {
		internalError(fail, err)
		return user, false
	}

	// Check the password.
	key, err := encryptPassword(password, salt)
	if err != nil {
		internalError(fail, err)
		return user, false
	}
	if !bytes.Equal(key, dbpassword) {
		fail(http.StatusForbidden)
		return user, false
	}
	return user, true
}

// authenticateRequest checks that the given user has permission to complete
// the request.
func authenticateRequest(request *http.Request, resource Resource) (ok bool) {
	return ((request.Method == http.MethodGet) && (resource.Permissions()&Read != 0)) || ((request.Method == http.MethodPost) && (resource.Permissions()&Write != 0))
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

	// Authenticate the user.
	user, ok := authenticateUser(fail, request, db)
	if !ok {
		return
	}

	// Get the corresponding resource and authenticate the request.
	resource, err := FromURI(user, request.URL.Path, db)
	if err == InvalidResource {
		http.NotFound(writer, request)
		return
	} else if err != nil {
		internalError(fail, err)
		return
	}
	if !authenticateRequest(request, resource) {
		fail(http.StatusForbidden)
		return
	}

	// Respond.
	switch request.Method {
	case http.MethodGet:
		// Write a response.
		enc := json.NewEncoder(writer)
		enc.SetEscapeHTML(true)
		err = resource.Write(enc)
		if err != nil {
			internalError(fail, err)
		}
	default:
		fail(http.StatusMethodNotAllowed)
	}
}

func main() {
	fmt.Printf("Starting server on :8080...\n")
	http.ListenAndServe(":8080", http.HandlerFunc(handle))
}

// vim: sw=4 ts=4 noexpandtab
