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

	_, err = db.Exec(`create table LocationIdToName(locationId integer, name text);
		insert into LocationIdToName values
		(1, "Melkroboter 1"),
		(2, "Liegebox NL"),
		(3, "Melkroboterbereich"),
		(4, "Melkroboter 2"),
		(5, "Liegebox HL"),
		(6, "Melkroboter 3"),
		(7, "Melkroboter 4"),
		(8, "Fressbereich NL"),
		(9, "Fressbereich HL")`)
	check(err)

	_, err = db.Exec(`create table GateIdToName(gateId int, name text);
		insert into GateIdToName values
		(1, "Gate NL"),
		(2, "Gate HL"),
		(3, "Gate Ausgang Melkbereich")`)
	check(err)

	_, err = db.Exec(`create view selections as
	select "Time", CowNr, GateIdToName.name as GateName, Origin.Name as Origin, Dst.Name as Dst
	from Sortings
	join LocationIdToName as Dst on Sortings.SortDst = Dst.locationId
	join LocationIdToName as Origin on Sortings.SortOri = Origin.locationId
	join GateIdToName on Sortings.Gate=GateIdToName.gateId`)
	check(err)

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
		 VALUES (?,?,?,?,?,?,?)`, se.Index, se.Time.Format("2006-01-02 15:04:05"), se.Transponder, se.CowName, se.SortSrc.Id, se.SortDst.Id, se.Gate.Id)
	check(err)
	err = tx.Commit()
	check(err)
}
