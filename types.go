package main

import (
	"database/sql"
)

type amputatorBot struct {
	dbConnected  bool
	dbConnection *sql.DB
	id           int
	stats        map[string]int
}
