/*
Tests for resource abstractions.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
    "fmt"
    "testing"
    "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type MockEncoder struct {
    contents []string
}

func (e *MockEncoder) Encode(item interface{}) error {
    e.contents = append(e.contents, fmt.Sprintf("%v", item))
    return nil
}

func TestProjectListPermissions(t *testing.T) {
    l := projectList{"", nil}
    if l.Permissions() != Get | Set | Create {
        t.Errorf("Project list should have all permissions!")
    }
}

func TestProjectListGet(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("opening database: %s", err)
    }

    mock.ExpectQuery(`SELECT .* FROM views WHERE name=?`).WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow("0").AddRow("1"))
    mock.ExpectQuery(`SELECT .* FROM owns WHERE name=?`).WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow("2"))

    l := projectList{"test", db}
    e := MockEncoder{[]string{}}
    err = l.Get(&e)
    if err != nil {
        t.Errorf("Unexpected error %q", err)
    }
    if len(e.contents) != 3 || e.contents[0] != "0" || e.contents[1] != "1" || e.contents[2] != "2" {
        t.Errorf("Expected '0 1 2', got %q", e.contents)
    }
    err = mock.ExpectationsWereMet()
    if err != nil {
        t.Errorf("Expectations were not met: %q", err)
    }
}
