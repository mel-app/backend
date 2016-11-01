/*	MEL app backend

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
func MELHandler(writer http.ResponseWriter, request *http.Request) {

	fmt.Printf("Handling request for %q\n", html.EscapeString(request.URL.Path))

	name, password, ok := request.BasicAuth()
	if !ok {
		writer.WriteHeader(401) // Auth required
		return
	}
	if name != "" || password != "" {
		// TODO: Implement checking against the db.
		writer.WriteHeader(403) // Invalid auth
		return
	}

	fmt.Fprintf(writer, "%q: authenticated as %s\n",
		html.EscapeString(request.URL.Path), name)
}

func main() {
	http.ListenAndServe(":8080", http.HandlerFunc(MELHandler))
}

// vim: sw=4 ts=4 noexpandtab
