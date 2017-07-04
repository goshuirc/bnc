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
	//TODO(dan): Warn before removing old db
	os.Remove(path)
	db := OpenDB(path)
	defer db.Close()
	_, err := db.Exec(`
CREATE TABLE ircbnc (
	key TEXT NOT NULL UNIQUE,
	value TEXT
);

INSERT INTO ircbnc (key, value) VALUES ("db_version", ?);

CREATE TABLE users (
	id TEXT NOT NULL UNIQUE,
	salt TEXT NOT NULL,
	password TEXT NOT NULL,
	default_nickname TEXT,
	default_fallback_nickname TEXT,
	default_username TEXT,
	default_realname TEXT
);

CREATE TABLE user_permissions (
	user_id TEXT NOT NULL,
	permission TEXT NOT NULL,
	FOREIGN KEY(user_id) REFERENCES users(id),
	PRIMARY KEY(user_id, permission)
);

CREATE TABLE server_connections (
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	nickname TEXT DEFAULT "",
	fallback_nickname TEXT DEFAULT "",
	username TEXT DEFAULT "",
	realname TEXT DEFAULT "",
	password TEXT DEFAULT "",
	FOREIGN KEY(user_id) REFERENCES users(id),
	PRIMARY KEY(user_id, name)
);

CREATE TABLE server_connection_accepted_certs (
	user_id TEXT NOT NULL,
	sc_name TEXT NOT NULL,
	cert TEXT NOT NULL,
	FOREIGN KEY(user_id, sc_name) REFERENCES server_connections(user_id, name)
);

CREATE TABLE server_connection_addresses (
	user_id TEXT NOT NULL,
	sc_name TEXT NOT NULL,
	address TEXT NOT NULL,
	port INTEGER,
	use_tls BOOL,
	FOREIGN KEY(user_id, sc_name) REFERENCES server_connections(user_id, name)
);

CREATE TABLE server_connection_channels (
	user_id TEXT NOT NULL,
	sc_name TEXT NOT NULL,
	name TEXT NOT NULL,
	key TEXT DEFAULT "",
	FOREIGN KEY(user_id, sc_name) REFERENCES server_connections(user_id, name),
	PRIMARY KEY(user_id, sc_name, name)
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
