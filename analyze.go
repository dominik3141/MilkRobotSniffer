package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	_ "github.com/mattn/go-sqlite3"
)

type Stay struct {
	CowNr    int16
	Begin    time.Time
	End      time.Time
	Location BarnLocation
}

func (st Stay) MarshalBinary() ([]byte, error) {
	bytes, err := json.Marshal(st)

	return bytes, err
}

func SortingResultsToStays(seIn <-chan SortEvent, stOut chan<- Stay) {
	// get connection to redisDB
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// keep the most recent stay and the most recent SortEvent of each cow in memory
	var cowToLastStay map[int16]*Stay
	if *useRedisBackup {
		val, err := rdb.Get("cowToLastStay").Bytes()
		check(err)
		err = json.Unmarshal(val, &cowToLastStay)
		check(err)
	} else {
		cowToLastStay = make(map[int16]*Stay)
	}

	for {
		// backup cowToLastStay map to redis
		rdb.Set("cowToLastStay", cowToLastStay, 0)

		cowsInMilkingArea := getCowsInMilkingArea(&cowToLastStay)
		fmt.Println("Cows in milkingArea:", len(cowsInMilkingArea))
		// save list of cows in milkingArea to redis
		// this is not the same as the backup of the cowToLastStay map!
		cowsInMilkingAreaJson, err := json.MarshalIndent(cowsInMilkingArea, "", "\t")
		check(err)
		err = rdb.Set("cowsInMilkingArea", cowsInMilkingAreaJson, 0).Err()
		check(err)

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

func getCowsInMilkingArea(cowToLastStay *map[int16]*Stay) []int16 {
	cowsInMilkingArea := make([]int16, 0, 16)

	for _, st := range *cowToLastStay {
		if st.Location.Id == 3 {
			fmt.Printf("%v is in milkingArea since %v\n", st.CowNr, st.Begin)
			cowsInMilkingArea = append(cowsInMilkingArea, st.CowNr)
		}
	}

	return cowsInMilkingArea
}

func (st Stay) Duration() time.Duration {
	return st.End.Sub(st.Begin)
}

func ShowStay(st Stay) {
	fmt.Printf("%v\t%v\t%v\t%v\n", st.Begin.Format("2006-01-02 15:04:05"), st.CowNr, st.Location.Name, st.Duration())
}
