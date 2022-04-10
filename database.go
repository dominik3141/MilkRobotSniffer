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
		TimeUnix INTEGER NOT NULL,
		"Time" TEXT NOT NULL,
		Transponder INTEGER NOT NULL,
		CowNr INTEGER NOT NULL,
		SortOri INTEGER NOT NULL,
		SortDst INTEGER NOT NULL,
		PictureObjName TEXT,
		Gate INTEGER NOT NULL)`)
	check(err)

	_, err = db.Exec(`create table LocationIdToName(locationId integer, name text);
		insert into LocationIdToName values
		(1, "Melkroboter 1"),
		(2, "Liegebox NL"),
		(3, "Melkbereich"),
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
		(3, "Gate Ausgang Melkbereich"),
		(4, "Melkroboter 1"),
		(5, "Melkroboter 2"),
		(6, "Melkroboter 3"),
		(7, "Melkroboter 4")`)
	check(err)

	_, err = db.Exec(`create view selections as
	select id, "Time", CowNr, GateIdToName.name as GateName, Origin.Name as Origin, Dst.Name as Dst
	from Sortings
	left join LocationIdToName as Dst on Sortings.SortDst = Dst.locationId
	left join LocationIdToName as Origin on Sortings.SortOri = Origin.locationId
	left join GateIdToName on Sortings.Gate=GateIdToName.gateId`)
	check(err)

	_, err = db.Exec(`create table Stays
		(Id INTEGER PRIMARY KEY AUTOINCREMENT,
		Inserted TEXT NOT NULL DEFAULT current_timestamp,
		Begin TEXT NOT NULL,
		End TEXT NOT NULL,
		CowNr INT NOT NULL,
		Duration INT NOT NULL,
		Location INT NOT NULL)`)
	check(err)

	_, err = db.Exec(`create view milkings as 
		select Stays.Begin, Stays.End, CowNr, Stays.Duration/60 as DurationMinutes, LocationIdToName.name as LocationName
		from Stays
		left join LocationIdToName on Location=LocationIdToName.locationId
		where Location=3
		order by Id`)
	check(err)

	_, err = db.Exec(`CREATE VIEW gateAverage as
		with tempTable as (
		select count(id) as cnt, GateName, CowNr
		from selections 
		group by GateName, CowNr
		--having cnt > 1
		having datetime("time") > datetime("now", "-24 hours")
		order by CowNr
		)

		select Gatename, avg(cnt) as av
		from tempTable
		group by Gatename`)
	check(err)

	db.Close()
}

func openDb(dbName string) *sql.DB {
	// don't forget to close the db later!
	db, err := sql.Open("sqlite3", dbName)
	check(err)

	return db
}

func insertSortEvent(se SortEvent, pictureObjName string, db *sql.DB) {
	tx, err := db.Begin()
	check(err)
	_, err = tx.Exec(`INSERT INTO Sortings("Index",TimeUnix, "Time", Transponder, CowNr, SortOri, SortDst, Gate, pictureObjName)
		 VALUES (?,?,?,?,?,?,?,?,?)`, se.Index, se.Time.Unix(), se.Time.Format("2006-01-02 15:04:05"), se.Transponder, se.CowName, se.SortSrc.Id, se.SortDst.Id, se.Gate.Id, pictureObjName)
	check(err)
	err = tx.Commit()
	check(err)
}

func insertStay(stay Stay, db *sql.DB) {
	tx, err := db.Begin()
	check(err)

	_, err = tx.Exec(`INSERT INTO Stays(Begin, End,CowNr, Duration, Location)
		 VALUES (?,?,?,?,?)`, stay.Begin.Format("2006-01-02 15:04:05"), stay.End.Format("2006-01-02 15:04:05"), stay.CowNr, int(stay.Duration().Seconds()), stay.Location.Id)
	check(err)

	err = tx.Commit()
	check(err)
}
