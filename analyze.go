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

// type recentLocation struct {
// 	CurrStay Stay
// 	LastSE   SortEvent
// 	CurrSE   SortEvent
// 	NextSE   SortEvent
// 	Wait     bool
// }

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

// func SortingResultsToStays(seIn <-chan SortEvent, stOut chan<- Stay) {
// 	// keep the most recent stay and the most recent SortEvent of each cow in memory
// 	cowToLastStay := make(map[int16]*recentLocation)

// 	for {
// 		// get a new sortEvent from channel
// 		se := <-seIn

// 		if se.DstIsRobo || (se.SortDst.Id != 3 && se.SortSrc.Id != 3) { // ignore sortings that have no connection to the waitingArea
// 			continue
// 		}

// 		rL, found := cowToLastStay[se.CowName]
// 		if !found {
// 			rL = new(recentLocation)

// 			var stay Stay
// 			stay.Begin = se.Time
// 			stay.CowNr = se.CowName
// 			stay.Location = se.SortDst

// 			rL.NextSE = se
// 			rL.CurrStay = stay
// 			rL.Wait = true

// 			cowToLastStay[se.CowName] = rL

// 			continue
// 		}

// 		if rL.Wait {
// 			rL.CurrSE = rL.NextSE
// 			rL.NextSE = se
// 			rL.Wait = false
// 			continue
// 		}

// 		rL.LastSE = rL.CurrSE
// 		rL.CurrSE = rL.NextSE
// 		rL.NextSE = se

// 		se = rL.CurrSE
// 		if se.SortDst.Id != rL.LastSE.SortDst.Id {

// 			if rL.NextSE.SortSrc.Id == se.SortSrc.Id {
// 				rL.CurrStay.End = rL.NextSE.Time
// 			} else {
// 				rL.CurrStay.End = se.Time
// 			}

// 			if se.SortSrc.Id != rL.CurrStay.Location.Id {
// 				rL.CurrStay.Problem = true
// 			}

// 			// save last stay before overwriting
// 			stOut <- rL.CurrStay

// 			var stay Stay
// 			stay.CowNr = se.CowName
// 			stay.Begin = se.Time
// 			stay.Location = se.SortDst

// 			// overwrite most recent stay
// 			rL.CurrStay = stay
// 		} else { // so se.SortDst.Id == rL.LastSE.SortDst.Id
// 			rL.CurrStay.Begin = se.Time
// 		}
// 	}
// }

func (st Stay) Duration() time.Duration {
	return st.End.Sub(st.Begin)
}

func ShowStay(st Stay) {
	fmt.Printf("%v\t%v\t%v\t%v\n", st.Begin.Format("2006-01-02 15:04:05"), st.CowNr, st.Location.Name, st.Duration())
}
