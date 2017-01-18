/*
Database interface code.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package backend

import (
	"database/sql"
	"log"
)

type DB struct {
	db *sql.DB
}

func NewDB(db *sql.DB) DB {
	return DB{db}
}

// Init clears and initialises the database with the expected tables.
func (d *DB) Init() {
	exec := []string{
		`DROP TABLE views`,
		`DROP TABLE owns`,
		`DROP TABLE deliverables`,
		`DROP TABLE projects`,
		`DROP TABLE users`,
		`CREATE TABLE users (
			name VARCHAR(320) PRIMARY KEY, -- 320 is the maximum email length.
			salt BYTEA,
			password BYTEA, -- Password is salted and encrypted.
			is_manager BOOL -- True if the user is also a manager.
		)`,
		`CREATE TABLE projects (
			id BIGINT PRIMARY KEY, -- Is this required??
			name VARCHAR(128), -- Type??
			percentage SMALLINT CHECK (percentage >= 0 and percentage <= 100),
			description VARCHAR(512), -- Size??
			updated DATE,
			version INT,
			flag BOOL,
			flag_version INT
		)`,
		`CREATE TABLE deliverables (
			id BIGINT,
			pid BIGINT,
			name VARCHAR(128),
			due DATE,
			percentage SMALLINT CHECK (percentage >= 0 and percentage <= 100),
			submitted BOOL, -- Whether or not the project is submitted.
			description VARCHAR(512), -- Size??
			updated DATE,
			version INT,
			PRIMARY KEY (id, pid)
		)`,
		`CREATE TABLE owns (
			name VARCHAR(320) REFERENCES users,
			pid BIGINT REFERENCES projects,
			PRIMARY KEY (name, pid)
		)`,
		`CREATE TABLE views (
			name VARCHAR(320) REFERENCES users,
			pid BIGINT REFERENCES projects,
			PRIMARY KEY (name, pid)
		)`,
		// Add a couple of test projects.
		`INSERT INTO projects VALUES (0, 'Test Project 0', 30, 'First test project', '1/17/2017', 0, TRUE, 0)`,
		`INSERT INTO projects VALUES (1, 'Test Project 1', 80, 'Second test project', '1/17/2017', 0, FALSE, 0)`,
		`INSERT INTO deliverables VALUES
			(0, 0, 'Deliverable 0', '11/25/2016', 20, FALSE, 'Finish backend', '1/17/2017', 0)`,
		`INSERT INTO deliverables VALUES
			(1, 0, 'Deliverable 1', '12/9/2016', 70, FALSE, 'Finish prototype', '1/17/2017', 0)`,
		// Add a test user.
		`INSERT INTO users VALUES ('test', '', '', TRUE)`,
		`INSERT INTO owns VALUES ('test', 0)`,
		`INSERT INTO views VALUES ('test', 1)`,
	}

	for _, cmd := range exec {
		_, err := d.db.Exec(cmd)
		if err != nil {
			log.Printf("Error executing '%s': %q\n", cmd, err)
		}
	}
}

// vim: sw=4 ts=4 noexpandtab
