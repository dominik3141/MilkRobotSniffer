package main

import (
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

func getStays(seIn) []Stay {
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
