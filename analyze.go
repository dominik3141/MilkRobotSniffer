package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Stay struct {
	CowNr    int
	Begin    time.Time
	End      time.Time
	Location int
}

// func main() {
// 	stays := getAllStays()
// 	insertStays(stays)
// }

func insertStays(stays []Stay) {
	db, err := sql.Open("sqlite3", "testdb01.db")
	check(err)
	defer db.Close()

	tx, err := db.Begin()
	check(err)
	for i, _ := range stays {
		_, err = tx.Exec(`INSERT INTO Stays(Begin, End, Duration, Location)
		 VALUES (?,?,?,?)`, stays[i].Begin, stays[i].End, stays[i].Duration(), stays[i].Location)
		check(err)
	}
	err = tx.Commit()
	check(err)
}

func getAllStays() []Stay {
	sortings := getAllSortings()

	stays := make([]Stay, 0)
	var stay Stay
	for i, se := range sortings {
		if i == 0 {
			stay.Begin = se.Time
			stay.Location = se.SortDst
			stays = append(stays, stay)
		} else {
			if se.SortDst != stays[len(stays)-1].Location {
				stays[len(stays)-1].End = se.Time
				stay.Begin = se.Time
				stay.Location = se.SortDst
				stays = append(stays, stay)
			} else {
				stays[len(stays)-1].Begin = se.Time
			}
		}
	}

	return stays
}

func getStays(cowName int) []Stay {
	sortings := getSortings(cowName)

	stays := make([]Stay, 0)
	var stay Stay
	for i, se := range sortings {
		if i == 0 {
			stay.Begin = se.Time
			stay.Location = se.SortDst
			stays = append(stays, stay)
		} else {
			if se.SortDst != stays[len(stays)-1].Location {
				stays[len(stays)-1].End = se.Time
				stay.Begin = se.Time
				stay.Location = se.SortDst
				stays = append(stays, stay)
			} else {
				stays[len(stays)-1].Begin = se.Time
			}
		}
	}

	return stays
}

func (st Stay) Duration() time.Duration {
	return st.End.Sub(st.Begin)
}

func ShowStays(stays []Stay) {
	BarnLocationIdToName := map[int]string{
		1: "Melkroboter 1",
		2: "Liegebox NL",
		3: "Melkbereich",
		4: "Melkroboter 2",
		5: "Liegebox HL",
		6: "Melkroboter 3",
		7: "Melkroboter 4",
		8: "Fressbereich NL",
		9: "Fressbereich HL",
	}

	for i, _ := range stays {
		if i > 0 && i < len(stays)-1 {
			if stays[i].Location == 3 {
				fmt.Println(stays[i].Begin.Format("2006-01-02 15:04:05"), BarnLocationIdToName[stays[i].Location], stays[i].Duration())
			}
		}
	}
}

// func getSortings(cowName int) []SortEvent {
// 	dbName := "testdb01.db"
// 	db, err := sql.Open("sqlite3", dbName)
// 	check(err)
// 	defer db.Close()

// 	rows, err := db.Query("select Id, TimeUnixMilli, CowNr, Gate, SortOri, SortDst from sortings where CowNr=? order by TimeUnixMilli", cowName)
// 	check(err)

// 	sortings := make([]SortEvent, 0, 128)
// 	var se SortEvent
// 	for rows.Next() {
// 		var ts int64
// 		err = rows.Scan(&se.Id, &ts, &se.CowNr, &se.Gate, &se.SortOri, &se.SortDst)
// 		check(err)
// 		se.Time = time.Unix(ts/1e3, 0)

// 		sortings = append(sortings, se)
// 	}

// 	return sortings
// }

// func getAllSortings() []SortEvent {
// 	dbName := "testdb01.db"
// 	db, err := sql.Open("sqlite3", dbName)
// 	check(err)
// 	defer db.Close()

// 	rows, err := db.Query("select Id, TimeUnixMilli, CowNr, Gate, SortOri, SortDst from sortings order by TimeUnixMilli")
// 	check(err)

// 	sortings := make([]SortEvent, 0, 128)
// 	var se SortEvent
// 	for rows.Next() {
// 		var ts int64
// 		err = rows.Scan(&se.Id, &ts, &se.CowNr, &se.Gate, &se.SortOri, &se.SortDst)
// 		check(err)
// 		se.Time = time.Unix(ts, 0)

// 		sortings = append(sortings, se)
// 	}

// 	return sortings
// }
