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

	"github.com/mel-app/backend/src"
)

var loginUrl = url + "login"
var newPassword = "2nd password"

var loginTests = []Test{
	// Authentication sanity checks.
	Test{
		Name:   "login:Unauthorized",
		Method: "GET", URL: loginUrl, Status: http.StatusUnauthorized,
		SetAuth: setNilAuth,
	},
	Test{
		Name:   "login:Forbidden",
		Method: "GET", URL: loginUrl, Status: http.StatusForbidden,
	},

	// Basic account functionality.
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
		SetAuth: setNewPassword,
	},
	Test{
		Name:   "login:MakeManager",
		Method: "GET", URL: loginUrl, Status: http.StatusOK,
		Pre:       makeManager,
		CheckBody: checkManager,
	},

	// Password change functionality.
	Test{
		Name:   "login:ChangePassword",
		Method: "PUT", URL: loginUrl, Status: http.StatusOK,
		Body: `{"Username":"` + defaultUser +
			`","Password":"` + newPassword + `","Manager":true}`,
	},
	Test{
		Name:   "login:TestNewPassword",
		Method: "GET", URL: loginUrl, Status: http.StatusOK,
		SetAuth:   setNewPassword,
		CheckBody: checkManager,
	},
	Test{
		Name:   "login:ResetPassword",
		Method: "PUT", URL: loginUrl, Status: http.StatusOK,
		SetAuth:   setNewPassword,
		Body: `{"Username":"` + defaultUser +
			`","Password":"` + defaultPassword + `","Manager":true}`,
	},
}

type login struct {
	User     string
	Password string
	Manager  bool
}

// setNilAuth does not set any auth.
func setNilAuth(r *http.Request) {
	return
}

// setNewPassword authenticates using the second password.
func setNewPassword(r *http.Request) {
	r.SetBasicAuth(defaultUser, newPassword)
}

// makeManager makes the default user a manager.
func makeManager(db *sql.DB) error {
	return backend.NewDB(db).SetIsManager(defaultUser, true)
}

// checkManager checks that the manager flag is set.
func checkManager(dec *json.Decoder) error {
	login := login{Manager: false}
	dec.Decode(&login)
	if login.Manager != true {
		return fmt.Errorf("Setting a manager did not work!")
	}
	return nil
}

// checkNotManager checks that the manager flag is not set.
func checkNotManager(dec *json.Decoder) error {
	login := login{Manager: true}
	dec.Decode(&login)
	if login.Manager != false {
		return fmt.Errorf("Default login is a manager!")
	}
	return nil
}

// vim: sw=4 ts=4 noexpandtab
