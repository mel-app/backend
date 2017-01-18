/*
Tests for the projects/ endpoint.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var projectsUrl = url + "projects"
var projectIds = []uint{}

var projectsTests = []Test{
	Test{
		Name:   "projects:Empty",
		Method: "GET", URL: projectsUrl, Status: http.StatusOK,
		CheckBody: checkIsEmpty,
	},
	Test{
		Name:   "projects:Create",
		Method: "POST", URL: projectsUrl, Status: http.StatusCreated,
		Body: `{"Name":"Test Project", "Updated":"2017-12-19"}`,
	},
	Test{
		Name:   "projects:GetList",
		Method: "GET", URL: projectsUrl, Status: http.StatusOK,
		CheckBody: func(dec *json.Decoder) error {
			err := getProjectIds(dec)
			if len(projectIds) != 1 {
				return fmt.Errorf("Expected a single project")
			}
			return err
		},
	},
}

// checkIsEmpty checks that the body is empty.
func checkIsEmpty(dec *json.Decoder) error {
	if dec.More() == true {
		return fmt.Errorf("Expected an empty project list")
	}
	return nil
}

// getProjectIds saves the list of project ids into the global projectIds.
func getProjectIds(dec *json.Decoder) error {
	for dec.More() {
		var i uint = 0
		err := dec.Decode(&i)
		if err != nil {
			return err
		}
		projectIds = append(projectIds, i)
	}
	return nil
}

// vim: sw=4 ts=4 noexpandtab
