package main

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func createDb(dbName string) {
	f, err := os.Create(dbName)
	check(err)
	defer f.Close()

	db, err := sql.Open("sqlite3", dbName)
	check(err)

	_, err = db.Exec(`CREATE TABLE Sortings
		(Id INTEGER PRIMARY KEY AUTOINCREMENT,
		Inserted TEXT NOT NULL DEFAULT current_timestamp,
		"Index" INTEGER NOT NULL,
		"Time" TEXT NOT NULL,
		Transponder INTEGER NOT NULL,
		CowNr INTEGER NOT NULL,
		SortOri INTEGER NOT NULL,
		SortDst INTEGER NOT NULL,
		Gate INTEGER NOT NULL)`)
	check(err)

	// create view ...

	// create id to name tables ...

	db.Close()
}

func openDb(dbName string) *sql.DB {
	// don't forget to close the db later!
	db, err := sql.Open("sqlite3", dbName)
	check(err)

	return db
}

func insertSortEvent(se SortEvent, db *sql.DB) {
	tx, err := db.Begin()
	check(err)
	_, err = tx.Exec(`INSERT INTO Sortings("Index", "Time", Transponder, CowNr, SortOri, SortDst, Gate)
		 VALUES (?,?,?,?,?,?,?)`, se.Index, se.Time, se.Transponder, se.CowName, se.SortSrc.Id, se.SortDst.Id, se.Gate.Id)
	check(err)
	err = tx.Commit()
	check(err)
}
