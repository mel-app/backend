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
	// Basic project functionality.
	Test{
		Name:   "projects:Empty",
		Method: "GET", URL: projectsUrl, Status: http.StatusOK,
		CheckBody: checkIsEmpty,
	},
	Test{
		Name:   "projects:Create",
		Method: "POST", URL: projectsUrl, Status: http.StatusCreated,
		BodyFunc: func() string {
			return `{"Name":"Test Project", "Updated":"2017-12-19"}`
		},
	},
	Test{
		Name:	"projects:CreateAsClientForbidden",
		Method: "POST", URL: projectsUrl,
		Status: http.StatusForbidden,
		SetAuth:	setClientAuth,
		BodyFunc: func() string {
			return `{"Name":"Test Project", "Updated":"2017-12-19"}`
		},
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
	Test{
		Name:	"projects:Get",
		Method:	"GET", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusOK,
		CheckBody: func(dec *json.Decoder) error {
			return checkProjectEqual(dec, project{
				Id: projectIds[0],
				Name: "Test Project",
				Owns: true,
			})
		},
	},
	Test{
		Name:   "projects:Put",
		Method:	"PUT", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusOK,
		BodyFunc: func() string {
			return fmt.Sprintf(`{"Name":"Test Project 2", "Updated":"2017-12-19", "Id":%d}`,
			projectIds[0])
		},
	},

	// Access from clients.
	Test{
		Name:	"projects:GetAsClientForbidden",
		Method:	"GET", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusForbidden,
		SetAuth:	setClientAuth,
	},
	// TODO: Implement adding/removing clients as part of "client" tests.
	Test{
		Name:	"clients:Add",
		Method:	"POST", URLFunc: func() string {
			return fmt.Sprintf("%s/%d/clients", projectsUrl, projectIds[0])
		},
		Status: http.StatusCreated,
		BodyFunc: func() string { return `{"Name":"` + client1User + `"}` },
	},
	Test{
		Name:	"projects:GetAsClient",
		Method:	"GET", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusOK,
		SetAuth:	setClientAuth,
		CheckBody: func(dec *json.Decoder) error {
			return checkProjectEqual(dec, project{
				Id: projectIds[0],
				Name: "Test Project 2",
			})
		},
	},
	Test{
		Name:   "projects:PutAsClientForbidden",
		Method:	"PUT", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status:	http.StatusForbidden,
		SetAuth:	setClientAuth,
		BodyFunc: func() string {
			return fmt.Sprintf(`{"Name":"Test Project 2", "Updated":"2017-12-19", "Id":%d}`,
			projectIds[0])
		},
	},

	// Deletion.
	Test{
		Name:	"projects:DeleteAsClient",
		Method:	"DELETE", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusOK,
		SetAuth:	setClientAuth,
	},
	Test{
		Name:	"projects:CheckDeleteAsClient",
		Method:	"GET", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusForbidden,
		SetAuth:	setClientAuth,
	},
	Test{
		Name:	"projects:CheckClientDeletionIsNotFull",
		Method:	"GET", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusOK,
		CheckBody: func(dec *json.Decoder) error {
			return checkProjectEqual(dec, project{
				Id: projectIds[0],
				Name: "Test Project 2",
				Owns: true,
			})
		},
	},
	Test{
		Name:	"clients:Add",
		Method:	"POST", URLFunc: func() string {
			return fmt.Sprintf("%s/%d/clients", projectsUrl, projectIds[0])
		},
		Status: http.StatusCreated,
		BodyFunc: func() string { return `{"Name":"` + client1User + `"}` },
	},
	Test{
		Name:	"projects:DeleteAsManager",
		Method:	"DELETE", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusOK,
	},
	Test{
		Name:	"projects:CheckDeletion",
		Method:	"GET", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusForbidden,
	},
	Test{
		Name:	"projects:CheckManagerDeletionIsFull",
		Method:	"GET", URLFunc: func() string {
			return fmt.Sprintf("%s/%d", projectsUrl, projectIds[0])
		},
		Status: http.StatusForbidden,
		SetAuth:	setClientAuth,
	},
}

type project struct {
	Id          uint
	Name        string
	Percentage  uint
	Description string
	Updated     string
	Version     uint
	Owns        bool
}

// checkIsEmpty checks that the body is empty.
func checkIsEmpty(dec *json.Decoder) error {
	if dec.More() == true {
		return fmt.Errorf("Expected an empty project list")
	}
	return nil
}

// checkProjectEqual checks that the project in the decoder is the same as
// the given project.
func checkProjectEqual(dec *json.Decoder, p project) error {
	json := project{}
	err := dec.Decode(&json)

	// TODO: We currently ignore the Updated date; compare using some other
	//		 method?
	json.Updated = ""

	if err != nil {
		return err
	}
	if p != json {
		return fmt.Errorf("%q != %q", p, json)
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
