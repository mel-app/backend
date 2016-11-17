/*
Tests for resource abstractions.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
    "database/sql"
    "fmt"
    "testing"
    "reflect"
    "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type MockEncoder struct {
    contents []string
}

func (e *MockEncoder) Encode(item interface{}) error {
    e.contents = append(e.contents, fmt.Sprintf("%v", item))
    return nil
}

type MockDecoder struct {
    contents []string
    cur int
}

func (d *MockDecoder) Decode(item interface{}) error {
    if d.cur < len(d.contents) {
        reflect.ValueOf(item).Elem().SetString(d.contents[d.cur])
        d.cur += 1
        return nil
    } else {
        return fmt.Errorf("Too many items decoded!")
    }
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

    mock.ExpectQuery("SELECT .* FROM views WHERE name=?").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow("0").AddRow("1"))
    mock.ExpectQuery("SELECT .* FROM owns WHERE name=?").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow("2"))

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

func TestProjectListSet(t *testing.T) {
    t.Skip("projectList Set is not yet implemented!")
}

func TestProjectListCreate(t *testing.T) {
    t.Skip("projectList Create is implemented elsewhere!")
}

func TestProjectPermissions(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("opening database: %s", err)
    }

    initDB := func(t *testing.T, views, owns, is_manager bool) {
        q := mock.ExpectQuery("SELECT pid FROM views WHERE .*").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow(0))
        if !views {
            q.WillReturnError(sql.ErrNoRows)
        }
        q = mock.ExpectQuery("SELECT pid FROM owns WHERE .*").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow(0))
        if !owns {
            q.WillReturnError(sql.ErrNoRows)
        }
        q = mock.ExpectQuery("SELECT is_manager FROM users WHERE name=?").WillReturnRows(sqlmock.NewRows([]string{"is_manager"}).AddRow(is_manager))
    }

    check := func(t *testing.T, expected int) {
        p, err := NewProject("test", 0, db)
        if err != nil {
            t.Fatalf("Unexpected error %q", err)
        }
        if p == nil {
            t.Fatalf("Returned project is unexpectedly nil!")
        }
        if p.Permissions() != expected {
            t.Errorf("Expected permissions %b, got %b", expected, p.Permissions())
        }
        err = mock.ExpectationsWereMet()
        if err != nil {
            t.Errorf("Expectations were not met: %q", err)
        }
    }

    t.Run("No project", func(t *testing.T) {
        initDB(t, false, false, false)
        check(t, 0)
    })
    t.Run("Manager", func(t *testing.T) {
        initDB(t, false, false, true)
        check(t, Create)
    })
    t.Run("Views", func(t *testing.T) {
        initDB(t, true, false, false)
        check(t, Get)
    })
    t.Run("Owns", func(t *testing.T) {
        initDB(t, false, true, false)
        check(t, Get | Set)
    })
    t.Run("Owns and is a manager", func(t *testing.T) {
        initDB(t, false, true, true)
        check(t, Get | Set | Create)
    })
}

func TestProjectGet(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("opening database: %s", err)
    }

    mock.ExpectQuery("SELECT .* FROM projects WHERE id=?").WillReturnRows(sqlmock.NewRows([]string{"name", "percentage", "description"}).AddRow("test proj", "10", "Desc"))

    p := project{0, 0, db, "test"}
    e := MockEncoder{[]string{}}
    err = p.Get(&e)
    if err != nil {
        t.Errorf("Unexpected error %q", err)
    }
    if len(e.contents) != 3 || e.contents[0] != "test proj" || e.contents[1] != "10" || e.contents[2] != "Desc" {
        t.Errorf("Expected 'test proj 10 Desc', got %q", e.contents)
    }
    err = mock.ExpectationsWereMet()
    if err != nil {
        t.Errorf("Expectations were not met: %q", err)
    }
}

func TestProjectSet(t *testing.T) {
    // TODO: Add test cases for synchronisation.
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("opening database: %s", err)
    }

    p := project{1, 0, db, "test"}

    check := func(t *testing.T, d MockDecoder, expErr error) {
        if expErr == nil {
            mock.ExpectExec("UPDATE projects SET .* WHERE id=.*").WithArgs("test proj", "10", "Desc", 1).WillReturnResult(sqlmock.NewResult(0, 0))
        }
        err := p.Set(&d)
        if err != expErr {
            t.Errorf("Expected error %v, got %v!", expErr, err)
        }
        err = mock.ExpectationsWereMet()
        if err != nil {
            t.Errorf("Expectations were not met: %q", err)
        }
    }

    t.Run("Empty body", func(t *testing.T) {
        check(t, MockDecoder{[]string{}, 0}, InvalidBody)
    })
    t.Run("Full body", func(t *testing.T) {
        check(t, MockDecoder{[]string{"test proj", "10", "Desc"}, 0}, nil)
    })
}
