/*
Main runner for the backend tests.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/mel-app/backend/src"
)

type Test struct {
	Name string
	Test func(*sql.DB) error
}

var port = "8080"
var url = "http://localhost:" + port + "/"

func main() {
	// Open the test database.
	dbname := os.Getenv("DATABASE_URL")
	if dbname == "" {
		dbname = "postgres://localhost/backend-test?sslmode=disable"
	}
	db, err := sql.Open("postgres", dbname)
	if err != nil {
		fmt.Printf("Error opening DB: %q\n", err)
		return
	}
	defer db.Close()

	// Start the backend in the background.
	go backend.Run(port, db)

	// Clear, initialise the test database.
	backendDB := backend.NewDB(db)
	backendDB.Init()

	// Suppress logging.
	log.SetOutput(ioutil.Discard)

	// Run the tests.
	run_tests(db)
}

// run_tests runs all the implemented tests.
func run_tests(db *sql.DB) {
	tests := [][]Test{
		loginTests,
	}

	for _, testSet := range tests {
		for _, test := range testSet {
			fmt.Printf("Running %s - ", test.Name)
			err := test.Test(db)
			if err == nil {
				fmt.Printf("ok\n")
			} else {
				fmt.Printf("%q\n", err)
			}
		}
	}
}

// checkStatus returns an error if the status code is not what was expected.
func checkStatus(response *http.Response, expected int) error {
	if response.StatusCode != expected {
		return fmt.Errorf("Expected %d, got %s!", expected, response.Status)
	}
	return nil
}

// vim: sw=4 ts=4 noexpandtab
