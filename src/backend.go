/*
MEL app backend.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package backend

import (
	"encoding/json"
	"log"
	"net/http"

	"database/sql"
)

// handle a single HTTP request.
func handle(writer http.ResponseWriter, request *http.Request, db *sql.DB) {
	// Wrapper for failing functions.
	fail := func(status int) { http.Error(writer, http.StatusText(status), status) }

	// Authenticate the user.
	user, password, ok := authenticateUser(writer, fail, request, db)
	if !ok {
		return
	}

	// get the corresponding defaultResource and authenticate the request.
	defaultResource, err := fromURI(user, password, request.URL.Path, db)
	if err == invalidResource {
		http.NotFound(writer, request)
		return
	} else if err != nil {
		internalError(fail, err)
		return
	}
	if !authenticateRequest(request, defaultResource) {
		fail(http.StatusForbidden)
		return
	}

	// Respond.
	enc := json.NewEncoder(writer)
	enc.SetEscapeHTML(true)
	switch request.Method {
	case http.MethodGet:
		err = defaultResource.get(enc)
	case http.MethodPut:
		err = defaultResource.set(json.NewDecoder(request.Body))
	case http.MethodPost:
		// Posts need to return 201 with a Location header with the URI to the
		// newly created defaultResource.
		// They should also use enc to write a representation of the object
		// created, preferably including the id.
		err = defaultResource.create(json.NewDecoder(request.Body),
			func(location string, item interface{}) error {
				writer.Header().Add("Location", location)
				writer.WriteHeader(http.StatusCreated)
				return enc.Encode(item)
			})
	case http.MethodDelete:
		err = defaultResource.delete()
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

// Run the server on the given port, connecting to the given database.
func Run(port string, db *sql.DB) {
	log.Printf("Running on port :%s\n", port)
	seed()
	log.Fatal(http.ListenAndServe(":"+port,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handle(w, r, db)
		}),
	))
}

// vim: sw=4 ts=4 noexpandtab
