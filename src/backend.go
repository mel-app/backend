/*
MEL app backend.



Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package backend

import (
	"encoding/json"
	"net/http"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

// TODO: Should be the actual db, ...
const dbtype = "sqlite3"
const dbname = "test.db"

// handle a single HTTP request.
func handle(writer http.ResponseWriter, request *http.Request) {
	// Wrapper for failing functions.
	fail := func(status int) { http.Error(writer, http.StatusText(status), status) }

	// Open the database.
	db, err := sql.Open(dbtype, dbname)
	if err != nil {
		internalError(fail, err)
		return
	}

	// Authenticate the user.
	user, ok := authenticateUser(writer, fail, request, db)
	if !ok {
		return
	}

	// get the corresponding resource and authenticate the request.
	resource, err := FromURI(user, request.URL.Path, db)
	if err == invalidResource {
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
	enc := json.NewEncoder(writer)
	enc.SetEscapeHTML(true)
	switch request.Method {
	case http.MethodGet:
		err = resource.get(enc)
	case http.MethodPut:
		err = resource.set(json.NewDecoder(request.Body))
	case http.MethodPost:
		// Posts need to return 201 with a Location header with the URI to the
		// newly created resource.
		// They should also use enc to write a representation of the object
		// created, preferably including the id.
		err = resource.create(json.NewDecoder(request.Body),
			func(location string, item interface{}) error {
				writer.Header().Add("Location", location)
				writer.WriteHeader(http.StatusCreated)
				return enc.Encode(item)
			})
	case http.MethodDelete:
		err = resource.delete()
	default:
		err = invalidMethod
	}
	if err == invalidBody {
		fail(http.StatusBadRequest)
	} else if err == invalidMethod {
		fail(http.StatusMethodNotAllowed)
	} else if err != nil {
		internalError(fail, err)
	}
}

func run(port string) {
	seed()
	http.ListenAndServe(port, http.HandlerFunc(handle))
}

// vim: sw=4 ts=4 noexpandtab
