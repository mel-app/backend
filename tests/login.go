/*
Tests for the login/ endpoint.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

var loginUrl = url + "login"
var user = "test user"
var password = "test user 2"

var loginTests = []Test{
	Test{"loginUnauthorized", loginUnauthorized},
	Test{"loginForbidden", loginUnauthorized},
	Test{"loginCreate", loginCreate},
	Test{"loginGet", loginGet},
	Test{"loginCreateAgain", loginCreateAgain},
}

type login struct {
	Manager bool
}

// loginUnauthorized checks that unauthorized access fails.
func loginUnauthorized(db *sql.DB) error {
	c := http.Client{}
	req, err := http.NewRequest("GET", loginUrl, nil)
	if err != nil {
		return err
	}
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkStatus(response, http.StatusUnauthorized)
}

// loginForbidden checks that access fails when wrong credentials are supplied.
func loginForbidden(db *sql.DB) error {
	c := http.Client{}
	req, err := http.NewRequest("GET", loginUrl, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, password)
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkStatus(response, http.StatusForbidden)
}

// loginCreate tests the login creation function.
func loginCreate(db *sql.DB) error {
	c := http.Client{}
	req, err := http.NewRequest("POST", loginUrl, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, password)
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkStatus(response, http.StatusCreated)
}

// loginGet tests that the login is actually created.
func loginGet(db *sql.DB) error {
	c := http.Client{}
	req, err := http.NewRequest("GET", loginUrl, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, password)
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	err = checkStatus(response, http.StatusOK)
	if err != nil {
		return err
	}
	login := login{true}
	json.NewDecoder(response.Body).Decode(&login)
	if login.Manager != false {
		return fmt.Errorf("Default login is a manager!")
	}
	return nil
}

// loginCreateAgain checks that trying to create the same user twice fails.
func loginCreateAgain(db *sql.DB) error {
	c := http.Client{}
	req, err := http.NewRequest("POST", loginUrl, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, "some other password")
	response, err := c.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkStatus(response, http.StatusForbidden)
}

// vim: sw=4 ts=4 noexpandtab
