/*
Main runner for the backend tests.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/mel-app/backend/src"
)

type Test struct {
	Name      string
	Pre       func(*sql.DB) error
	Post      func(*sql.DB) error
	Method    string
	URL       string
	Status    int
	Body      string
	CheckBody func(*json.Decoder) error
	SetAuth   func(*http.Request)
}

var defaultUser = "test user"
var defaultPassword = "test user 2"

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
	backend.NewDB(db).Init()

	// Suppress logging.
	log.SetOutput(ioutil.Discard)

	// Run the tests.
	runTests(db)
}

// runTests runs all the implemented tests.
func runTests(db *sql.DB) {
	tests := [][]Test{
		loginTests,
		projectsTests,
	}

	for _, testSet := range tests {
		for _, test := range testSet {
			fmt.Printf("Running %s - ", test.Name)
			err := runTest(test, db)
			if err == nil {
				fmt.Printf("ok\n")
			} else {
				fmt.Printf("%q\n", err)
			}
		}
	}
}

// runTest runs a single given Test.
func runTest(t Test, db *sql.DB) error {
	if t.Pre != nil {
		err := t.Pre(db)
		if err != nil {
			return err
		}
	}

	c := http.Client{}
	req, err := http.NewRequest(t.Method, t.URL,
		bytes.NewBufferString(t.Body))
	if err != nil {
		return err
	}

	if t.SetAuth != nil {
		t.SetAuth(req)
	} else {
		req.SetBasicAuth(defaultUser, defaultPassword)
	}

	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != t.Status {
		return fmt.Errorf("Expected %d, got %s!", t.Status, response.Status)
	}

	if t.CheckBody != nil {
		err = t.CheckBody(json.NewDecoder(response.Body))
		if err != nil {
			return err
		}
	}

	if t.Post != nil {
		err := t.Post(db)
		if err != nil {
			return err
		}
	}
	return nil
}

// vim: sw=4 ts=4 noexpandtab
