/*
MEL app backend.


 
Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"net/http"
	"fmt"
	"html"
)

// Handle a single HTTP request.
func Handle(writer http.ResponseWriter, request *http.Request) {

	fmt.Printf("Handling request for %q\n", html.EscapeString(request.URL.Path))

	// Authenticate.
	name, password, ok := request.BasicAuth()
	if !ok {
		http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if name != "" || password != "" {
		// TODO: Implement checking against the db.
		http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	fmt.Fprintf(writer, "%q: authenticated as %s\n",
		html.EscapeString(request.URL.Path), name)

	// Parse the URL and return the corresponding value.
}

func main() {
	http.ListenAndServe(":8080", http.HandlerFunc(Handle))
}

// vim: sw=4 ts=4 noexpandtab
