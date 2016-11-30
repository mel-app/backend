/*
Authentication code.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package backend

import (
	"bytes"
	"crypto/rand"
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
func authenticateUser(writer http.ResponseWriter, fail func(int), request *http.Request, db *sql.DB) (user string, ok bool) {
	// Get the user name and password.
	user, password, ok := request.BasicAuth()
	if !ok {
		writer.Header().Add("WWW-Authenticate", "basic realm=\"\"")
		fail(http.StatusUnauthorized)
		return user, false
	}

	// Retrieve the salt and database password.
	salt := make([]byte, passwordSize)
	dbpassword := []byte("")
	err := db.QueryRow("SELECT salt, password FROM users WHERE name=?", user).Scan(&salt, &dbpassword)
	if err == sql.ErrNoRows && request.URL.Path == "/login" && request.Method == http.MethodPost {
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
		_, err = db.Exec("INSERT INTO users VALUES (?, ?, ?, ?)", user, salt, key, false)
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
	if string(dbpassword) == "" {
		// Special case an empty password in the database.
		// This lets us create "public" demonstration accounts.
		return user, true
	}
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
	return ((request.Method == http.MethodGet) && (resource.Permissions()&Get != 0)) ||
		((request.Method == http.MethodPut) && (resource.Permissions()&Set != 0)) ||
		((request.Method == http.MethodPost) && (resource.Permissions()&Create != 0)) ||
		((request.Method == http.MethodDelete) && (resource.Permissions()&Delete != 0))
}

// vim: sw=4 ts=4 noexpandtab
