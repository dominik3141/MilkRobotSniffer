package main

import (
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Stay struct {
	CowNr    int16
	Begin    time.Time
	End      time.Time
	Location BarnLocation
}

func SortingResultsToStays(seIn <-chan SortEvent, stOut chan<- Stay) {
	// keep the most recent stay and the most recent SortEvent of each cow in memory
	cowToLastStay := make(map[int16]*Stay)

	for {
		// get a new sortEvent from channel
		se := <-seIn

		if se.DstIsRobo || (se.SortDst.Id != 3 && se.SortSrc.Id != 3) || (se.SortDst.Id == 3 && se.SortSrc.Id == 3) { // ignore sortings that have no connection to the waitingArea
			continue
		}

		stay, found := cowToLastStay[se.CowName]
		if !found {
			stay := new(Stay)
			stay.Begin = se.Time
			stay.CowNr = se.CowName
			stay.Location = se.SortDst

			cowToLastStay[se.CowName] = stay
			continue
		}

		if se.SortSrc.Id != stay.Location.Id {
			stay = new(Stay)
			stay.CowNr = se.CowName
			stay.Location = se.SortDst
			stay.Begin = se.Time
			cowToLastStay[se.CowName] = stay

			continue
		}

		if se.SortDst.Id == stay.Location.Id {
			stay.Begin = se.Time
			continue
		}

		stay.End = se.Time
		stOut <- *stay

		stay = new(Stay)
		stay.CowNr = se.CowName
		stay.Location = se.SortDst
		stay.Begin = se.Time
		cowToLastStay[se.CowName] = stay
	}
}

func (st Stay) Duration() time.Duration {
	return st.End.Sub(st.Begin)
}

func ShowStay(st Stay) {
	fmt.Printf("%v\t%v\t%v\t%v\n", st.Begin.Format("2006-01-02 15:04:05"), st.CowNr, st.Location.Name, st.Duration())
}
