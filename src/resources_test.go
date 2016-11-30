/*
Tests for resource abstractions.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package backend

import (
	"database/sql"
	"fmt"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"reflect"
	"testing"
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
	cur      int
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

func (d *MockDecoder) More() bool {
	return d.cur < len(d.contents)
}

type MockProjectDecoder struct {
	contents project
}

func (d *MockProjectDecoder) Decode(item interface{}) error {
	f := reflect.ValueOf(item).Elem()
	f.FieldByName("Pid").SetUint(uint64(d.contents.Pid))
	f.FieldByName("Name").SetString(d.contents.Name)
	f.FieldByName("Percentage").SetUint(uint64(d.contents.Percentage))
	f.FieldByName("Description").SetString(d.contents.Description)
	f.FieldByName("Owns").SetBool(d.contents.Owns)
	return nil
}

func (d *MockProjectDecoder) More() bool {
	return false
}

func TestProjectListPermissions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	initDB := func(t *testing.T, is_manager bool) {
		mock.ExpectQuery("SELECT is_manager FROM users WHERE name=?").
			WillReturnRows(sqlmock.NewRows([]string{"is_manager"}).
				AddRow(is_manager))
	}

	check := func(t *testing.T, expected int) {
		p, err := NewProjectList("test", db)
		if err != nil {
			t.Fatalf("Unexpected error %q", err)
		}
		if p == nil {
			t.Fatalf("Returned project list is unexpectedly nil!")
		}
		if p.Permissions() != expected {
			t.Errorf("Expected permissions %b, got %b", expected, p.Permissions())
		}
		err = mock.ExpectationsWereMet()
		if err != nil {
			t.Errorf("Expectations were not met: %q", err)
		}
	}

	t.Run("Not a manager", func(t *testing.T) {
		initDB(t, false)
		check(t, get)
	})
	t.Run("Manager", func(t *testing.T) {
		initDB(t, true)
		check(t, get|create)
	})
}

func TestProjectListget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	mock.ExpectQuery("SELECT .* FROM views WHERE name=?").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow("0").AddRow("1"))
	mock.ExpectQuery("SELECT .* FROM owns WHERE name=?").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow("2"))

	l := projectList{resource{}, "test", 0, db}
	e := MockEncoder{[]string{}}
	err = l.get(&e)
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

func TestProjectPermissions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	initDB := func(t *testing.T, views, owns bool) {
		q := mock.ExpectQuery("SELECT pid FROM views WHERE .*").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow(0))
		if !views {
			q.WillReturnError(sql.ErrNoRows)
		}
		q = mock.ExpectQuery("SELECT pid FROM owns WHERE .*").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow(0))
		if !owns {
			q.WillReturnError(sql.ErrNoRows)
		}
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
		initDB(t, false, false)
		check(t, 0)
	})
	t.Run("Views", func(t *testing.T) {
		initDB(t, true, false)
		check(t, get|delete)
	})
	t.Run("Owns", func(t *testing.T) {
		initDB(t, false, true)
		check(t, get|set|delete)
	})
}

func TestProjectget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	mock.ExpectQuery("SELECT .* FROM projects WHERE id=?").
		WillReturnRows(sqlmock.NewRows([]string{"name", "percentage",
			"description"}).AddRow("test proj", "10", "Desc"))

	p := projectResource{resource{}, 0, 0, db, "test"}
	e := MockEncoder{[]string{}}
	err = p.get(&e)
	if err != nil {
		t.Errorf("Unexpected error %q", err)
	}
	result := `{0 test proj 10 Desc false}`
	if len(e.contents) != 1 || e.contents[0] != result {
		t.Errorf("Expected '%s', got %q", result, e.contents)
	}
	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Errorf("Expectations were not met: %q", err)
	}
}

func TestProjectset(t *testing.T) {
	// TODO: Add test cases for synchronisation.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	p := projectResource{resource{}, 1, 0, db, "test"}

	check := func(t *testing.T, d decoder, expErr error) {
		if expErr == nil {
			mock.ExpectExec("UPDATE projects SET .* WHERE id=.*").
				WithArgs("test proj", 10, "Desc", 1).
				WillReturnResult(sqlmock.NewResult(0, 0))
		}
		err := p.set(d)
		if err != expErr {
			t.Errorf("Expected error %v, got %v!", expErr, err)
		}
		err = mock.ExpectationsWereMet()
		if err != nil {
			t.Errorf("Expectations were not met: %q", err)
		}
	}

	check(t, &MockProjectDecoder{project{1, "test proj", 10, "Desc", false}}, nil)
}

func TestClientsPermissions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	initDB := func(t *testing.T, views, owns bool) {
		q := mock.ExpectQuery("SELECT pid FROM views WHERE .*").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow(0))
		if !views {
			q.WillReturnError(sql.ErrNoRows)
		}
		q = mock.ExpectQuery("SELECT pid FROM owns WHERE .*").WillReturnRows(sqlmock.NewRows([]string{"pid"}).AddRow(0))
		if !owns {
			q.WillReturnError(sql.ErrNoRows)
		}
	}

	check := func(t *testing.T, expected int) {
		c, err := NewClients("test", 0, db)
		if err != nil {
			t.Fatalf("Unexpected error %q", err)
		}
		if c == nil {
			t.Fatalf("Returned clients is unexpectedly nil!")
		}
		if c.Permissions() != expected {
			t.Errorf("Expected permissions %b, got %b", expected, c.Permissions())
		}
		err = mock.ExpectationsWereMet()
		if err != nil {
			t.Errorf("Expectations were not met: %q", err)
		}
	}

	t.Run("Views", func(t *testing.T) {
		initDB(t, true, false)
		check(t, 0)
	})
	t.Run("Owns", func(t *testing.T) {
		initDB(t, false, true)
		check(t, get|set)
	})
}

func TestClientsset(t *testing.T) {
	// TODO: Add test cases for synchronisation.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	c := clients{resource{}, 1, nil, db}

	check := func(t *testing.T, update, existing []string) {
		rows := sqlmock.NewRows([]string{"name"})
		for _, v := range existing {
			rows.AddRow(v)
		}
		mock.ExpectQuery("SELECT name FROM views WHERE .*").
			WillReturnRows(rows).WithArgs(c.pid)

		// Look for added values.
		for _, s := range update {
			in := false
			for _, v := range existing {
				if v == s {
					in = true
				}
			}
			if !in {
				mock.ExpectQuery(`SELECT name FROM users WHERE name=\?`).
					WithArgs(s).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow(s))
				mock.ExpectExec("INSERT INTO views VALUES .*").WithArgs(s, c.pid).WillReturnResult(sqlmock.NewResult(0, 0))
			}
		}

		// Look for removed values.
		for _, s := range existing {
			in := false
			for _, v := range update {
				if v == s {
					in = true
				}
			}
			if !in {
				mock.ExpectExec("DELETE FROM views .*").WithArgs(s, c.pid).WillReturnResult(sqlmock.NewResult(0, 0))
			}
		}

		err := c.set(&MockDecoder{update, 0})
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		err = mock.ExpectationsWereMet()
		if err != nil {
			t.Errorf("Expectations were not met: %q", err)
		}
	}

	// We can only test single item changes here as sqlmock requires ordered
	// queries.
	t.Run("Empty", func(t *testing.T) {
		check(t, []string{}, []string{})
	})
	t.Run("Remove", func(t *testing.T) {
		check(t, []string{"2"}, []string{"1", "2"})
	})
	t.Run("Add", func(t *testing.T) {
		check(t, []string{"1", "2", "3"}, []string{"2", "3"})
	})
	t.Run("Remove and add", func(t *testing.T) {
		check(t, []string{"1", "2"}, []string{"2", "3"})
	})
}

type mockFlagDecoder struct {
	value flag
}

func (f *mockFlagDecoder) Decode(item interface{}) error {
	reflect.ValueOf(item).Elem().Set(reflect.ValueOf(f.value))
	return nil
}

func (f *mockFlagDecoder) More() bool {
	return false
}

func TestFlagset(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}

	f := flagResource{resource{}, 1, nil, db}

	check := func(name string, update, existing, result flag) {
		t.Run(name, func(t *testing.T) {

			// Expect for the existing value.
			row := sqlmock.NewRows([]string{"flag", "flag_version"}).
				AddRow(existing.Value, existing.Version)
			mock.ExpectQuery("SELECT flag, flag_version FROM projects WHERE .*").
				WillReturnRows(row).WithArgs(f.pid)

			// Expect for the result.
			if existing.Value != result.Value {
				mock.ExpectExec(`UPDATE projects SET flag=\?, flag_version=\? WHERE id=\?`).
					WillReturnResult(sqlmock.NewResult(0, 0)).
					WithArgs(result.Value, result.Version, f.pid)
			}

			err := f.set(&mockFlagDecoder{update})
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}
			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("Expectations were not met: %q", err)
			}
		})
	}

	check("No change", flag{2, false}, flag{2, false}, flag{2, false})
	check("No change but server version has been incremented", flag{2, false}, flag{4, false}, flag{4, false})
	check("Server updated", flag{2, false}, flag{3, true}, flag{3, true})
	check("Client updated", flag{2, true}, flag{2, false}, flag{3, true})
	check("Client and server updated", flag{2, true}, flag{4, false}, flag{4, false})
}

// vim: sw=4 ts=4 noexpandtab
