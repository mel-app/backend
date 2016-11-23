/*
MEL app utility.

This provides a executable for performing various simple actions which would
need extra authentication.

Author:		Alastair Hughes
Contact:	<hobbitalastair at yandex dot com>
*/

package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
)

// usage prints the usage string for the app.
func usage() {
	fmt.Printf("%s [bless <user>] | [password <user> <pass>] | [transfer <project> <user>] | [list <user>] | [serve [<port>]]\n", os.Args[0])
}

func main() {
	db, err := sql.Open(dbtype, dbname)
	if err != nil {
		fmt.Printf("Error opening DB: %q\n", err)
		return
	}

	if len(os.Args) < 2 {
		usage()
		return
	}

	if os.Args[1] == "bless" && len(os.Args) == 3 {
		bless(os.Args[2], db)
	} else if os.Args[1] == "password" && len(os.Args) == 4 {
		password(os.Args[2], os.Args[3], db)
	} else if os.Args[1] == "transfer" && len(os.Args) == 4 {
		transfer(os.Args[2], os.Args[3], db)
	} else if os.Args[1] == "list" && len(os.Args) == 3 {
		list(os.Args[2], db)
	} else if os.Args[1] == "serve" {
		port := "8080"
		if len(os.Args) == 3 {
			port = os.Args[2]
		} else if len(os.Args) != 2 {
			usage()
			return
		}
		fmt.Printf("Starting server on :%s...\n", port)
		run(":" + port)
	} else {
		usage()
	}
}

// bless marks a user as a manager
func bless(user string, db *sql.DB) {
	_, err := db.Exec("UPDATE users SET is_manager=? WHERE id=?", true, user)
	if err != nil {
		fmt.Printf("Error blessing user: %q\n", err)
	}
}

// password resets the given user's password
func password(user, password string, db *sql.DB) {
	salt := make([]byte, passwordSize)
	err := db.QueryRow("SELECT salt FROM users WHERE name=?", user).Scan(&salt)
	if err != nil {
		fmt.Printf("Failed to find existing salt: %q\n", err)
	}
	key, err := encryptPassword(password, salt)
	if err != nil {
		fmt.Printf("Failed to encrypt the user password: %q\n", err)
	}
	_, err = db.Exec("UPDATE users SET password=? WHERE name=?", key, user)
	if err != nil {
		fmt.Printf("Failed to update the user password: %q\n", err)
	}
}

// transfer sets a project's owner to "user"
func transfer(spid string, user string, db *sql.DB) {
	pid, err := strconv.Atoi(spid)
	if err != nil || pid < 0 {
		fmt.Printf("Invalid pid %s\n", spid)
	}
	_, err = db.Exec("UPDATE owns SET name=? WHERE pid=?", user, uint(pid))
	if err != nil {
		fmt.Printf("Failed to update the owner: %q\n", err)
	}
}

// list the user's projects with the corresponding PID
func list(user string, db *sql.DB) {
	rows, err := db.Query("SELECT pid FROM owns WHERE name=?", user)
	if err != nil {
		fmt.Printf("Error getting rows: %q\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		id := -1
		err = rows.Scan(&id)
		if err != nil {
			fmt.Printf("Error getting value: %q\n", err)
			return
		}
		name := ""
		err = db.QueryRow("SELECT name FROM projects WHERE id=?", id).
			Scan(&name)
		if err != nil {
			fmt.Printf("Failed to get the name for project %d: %q\n", id, err)
			return
		}
		fmt.Printf("%d: %s\n", id, name)
	}
	if rows.Err() != nil {
		fmt.Printf("Error getting more rows: %q\n", rows.Err())
	}
}

// vim: sw=4 ts=4 noexpandtab
