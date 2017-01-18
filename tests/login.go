/*
Tests for the login/ endpoint.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var loginUrl = url + "login"

var loginTests = []Test{
	Test{
		Name:   "login:Unauthorized",
		Method: "GET", URL: loginUrl, Status: http.StatusUnauthorized,
		SetAuth: setNilAuth,
	},
	Test{
		Name:   "login:Forbidden",
		Method: "GET", URL: loginUrl, Status: http.StatusForbidden,
	},
	Test{
		Name:   "login:Create",
		Method: "POST", URL: loginUrl, Status: http.StatusCreated,
	},
	Test{
		Name:   "login:Get",
		Method: "GET", URL: loginUrl, Status: http.StatusOK,
		CheckBody: checkNotManager,
	},
	Test{
		Name:   "login:CreateAgain",
		Method: "POST", URL: loginUrl, Status: http.StatusForbidden,
		SetAuth: setWrongPassword,
	},
}

type login struct {
	Manager bool
}

// setNilAuth does not set any auth.
func setNilAuth(r *http.Request) {
	return
}

// setWrongPassword provides an invalid password.
func setWrongPassword(r *http.Request) {
	r.SetBasicAuth(defaultUser, "some other password")
}

// checkNotManager checks that the manager flag is not set.
func checkNotManager(dec *json.Decoder) error {
	login := login{true}
	dec.Decode(&login)
	if login.Manager != false {
		return fmt.Errorf("Default login is a manager!")
	}
	return nil
}

// vim: sw=4 ts=4 noexpandtab
