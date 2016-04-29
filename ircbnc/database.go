// This file is based on 'database.go' from Oragono/Ergonomadic
// it is modified by Daniel Oaks <daniel@danieloaks.net>
// covered by the MIT license in the LICENSE.ergonomadic file

package ircbnc

import (
	"database/sql"
	"log"
	"os"

	// db drivers should be imported anonymously
	_ "github.com/mattn/go-sqlite3"
)

// LatestDbVersion is the latest version of the the database
const LatestDbVersion = 1

// InitDB creates the new blank database
func InitDB(path string) {
	os.Remove(path)
	db := OpenDB(path)
	defer db.Close()
	_, err := db.Exec(`
CREATE TABLE gircbnc (
	key TEXT NOT NULL UNIQUE,
	value TEXT
);

INSERT INTO gircbnc (key, value) VALUES ("db_version", ?);

CREATE TABLE users (
	id TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	default_nickname TEXT,
	default_username TEXT,
	default_realname TEXT
);

CREATE TABLE server_connections (
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	nickname TEXT,
	username TEXT,
	realname TEXT,
	FOREIGN KEY(user_id) REFERENCES users(id),
	PRIMARY KEY(user_id, name)
);

CREATE TABLE server_connection_accepted_certs (
	user_id TEXT NOT NULL,
	server_connection_name TEXT NOT NULL,
	cert TEXT NOT NULL,
	FOREIGN KEY(user_id, server_connection_name) REFERENCES server_connections(user_id, name)
);

CREATE TABLE server_connection_addresses (
	user_id TEXT NOT NULL,
	server_connection_name TEXT NOT NULL,
	address TEXT NOT NULL,
	port INTEGER,
	use_ssl BOOL,
	FOREIGN KEY(user_id, server_connection_name) REFERENCES server_connections(user_id, name)
);`, LatestDbVersion)
	if err != nil {
		log.Fatal("initdb error: ", err)
	}
}

// UpgradeDB upgrades the database to the latest version
func UpgradeDB(path string) {
	//db := OpenDB(path)

	//TODO(dan): Actually write upgrading code here
}

// OpenDB returns a database handle
func OpenDB(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal("open db error: ", err)
	}
	return db
}
