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
	Problem  bool // set this if a strange thing has been detected, like a cow leaving a location it never entered
}

type recentLocation struct {
	IsFirstSE bool
	LastStay  Stay
	LastSE    SortEvent
}

func SortingResultsToStays(seIn <-chan SortEvent, stOut chan<- Stay) {
	// keep the most recent stay and the most recent SortEvent of each cow in memory
	cowToLastStay := make(map[int16]*recentLocation)

	for {
		// get a new sortEvent from channel
		se := <-seIn
		// nextSe := <- seIn

		if se.DstIsRobo || (se.SortDst.Id != 3 && se.SortSrc.Id != 3) { // ignore sortings that have no connection to the waitingArea
			continue
		}

		if se.SortDst.Id == 3 && se.SortSrc.Id == 3 { // ignore cows that try to exit the waitingArea but fail
			continue
		}

		rL, found := cowToLastStay[se.CowName]
		if !found {
			rL = new(recentLocation)
			rL.IsFirstSE = true
			cowToLastStay[se.CowName] = rL
		}

		var stay Stay
		if rL.IsFirstSE == true {
			stay.Begin = se.Time
			stay.CowNr = se.CowName
			stay.Location = se.SortDst

			rL.IsFirstSE = false
			rL.LastSE = se
			rL.LastStay = stay
		} else {
			if se.SortDst.Id != rL.LastSE.SortDst.Id {
				nextSe := <-seIn
				for nextSe.DstIsRobo || (se.SortDst.Id != 3 && se.SortSrc.Id != 3) {
					nextSe = <-seIn
				}

				if nextSe.SortSrc.Id == se.SortSrc.Id {
					rL.LastStay.End = nextSe.Time
				} else {
					rL.LastStay.End = se.Time
				}

				if se.SortSrc.Id != rL.LastStay.Location.Id {
					rL.LastStay.Problem = true
				}

				// save last stay before overwriting
				stOut <- rL.LastStay

				stay.CowNr = se.CowName
				stay.Begin = se.Time
				stay.Location = se.SortDst

				// overwrite most recent stay
				rL.LastStay = stay
			} else { // so se.SortDst.Id == rL.LastSE.SortDst.Id
				rL.LastStay.Begin = se.Time
			}
		}
	}
}

func (st Stay) Duration() time.Duration {
	return st.End.Sub(st.Begin)
}

func ShowStay(st Stay) {
	fmt.Printf("%v\t%v\t%v\t%v\n", st.Begin.Format("2006-01-02 15:04:05"), st.CowNr, st.Location.Name, st.Duration())
}
