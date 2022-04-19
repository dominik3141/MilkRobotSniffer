package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	// create a logger
	filename := "sortingToStay_" + fmt.Sprintf("%v", time.Now()) + ".log"
	f, err := os.Create(filename)
	check(err)
	defer f.Close()
	logger := log.New(f, "", log.LstdFlags)

	// get connection to redisDB
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// keep the most recent stay and the most recent SortEvent of each cow in memory
	// var cowToLastStay map[int16]*Stay
	// if *useRedisBackup {
	// 	val, err := rdb.Get("cowToLastStay").Bytes()
	// 	check(err)
	// 	err = json.Unmarshal(val, &cowToLastStay)
	// 	check(err)
	// } else {
	// 	cowToLastStay = make(map[int16]*Stay)
	// }

	registerCowInMA := func(cownr int16) {
		// add a cow to the set of cows that are inside the milkingArea

		err := rdb.SAdd("cowsInMA", cownr).Err()
		check(err)
	}

	removeCowFromMA := func(cownr int16) {
		// remove a cow from the set of cows which are inside the milkingArea

		err := rdb.SRem("cowsInMA", cownr).Err()
		check(err)
	}

	getLastStay := func(cownr int16) (Stay, bool) {
		val, err := rdb.Get(fmt.Sprint(cownr)).Bytes()
		if err != nil && err != redis.Nil {
			check(err)
			return Stay{}, false
		} else if err == redis.Nil {
			return Stay{}, false
		} else {
			var st Stay
			err = json.Unmarshal(val, &st)
			return st, true
		}
	}

	setLastStay := func(cownr int16, val Stay) {
		err := rdb.Set(fmt.Sprint(cownr), val, 0).Err() // we already have a binaryMarshaler for the Stay type, so nothing to do here
		check(err)
	}

	for {
		// save list of cows in milkingArea to redis
		// this is not the same as the backup of the cowToLastStay map!
		// cowsInMilkingArea := getCowsInMilkingArea(&cowToLastStay)
		// fmt.Println("Cows in milkingArea:", len(cowsInMilkingArea))
		// cowsInMilkingAreaJson, err := json.MarshalIndent(cowsInMilkingArea, "", "\t")
		// check(err)
		// err = rdb.Set("cowsInMilkingArea", cowsInMilkingAreaJson, 0).Err()
		// check(err)

		// get a new sortEvent from channel
		se := <-seIn

		if se.SortDst.Id == 3 && se.SortSrc.Id == 3 {
			// milkingArea -> milkingArea
			stay, found := getLastStay(se.CowName)
			if !found {
				continue
			}
			registerCowInMA(se.CowName)
			if stay.Location.Id != 3 {
				logger.Printf("%v\tStrange situation: Cow was seen by robo, but allegedly has not been in the milkingArea at that time. Cows alleged location: %v.\n", se.CowName, stay.Location.Name)
			}
			continue
		}

		if se.SortDst.Id != 3 && se.SortSrc.Id != 3 {
			// unrelated to the milkingArea
			// i.e. Liegebox NL -> Fressbereich NL
			stay, found := getLastStay(se.CowName)
			if !found {
				continue
			}
			removeCowFromMA(se.CowName)
			if stay.Location.Id == 3 {
				logger.Printf("%v\tStrange situation: The database thinks that our cow would be inside the milkingArea, but she was seen going from %v to %v.\n", se.CowName, se.SortSrc.Name, se.SortDst.Name)
			}
			continue
		}

		if se.DstIsRobo {
			// so that cow has to be in the milkingArea
			stay, found := getLastStay(se.CowName)
			if !found {
				continue
			}
			registerCowInMA(se.CowName)
			if stay.Location.Id != 3 {
				logger.Printf("%v\tStrange situation: Cow was seen by robo, but allegedly has not been in the milkingArea at that time. Cows alleged location: %v.\n", se.CowName, stay.Location.Name)
			}
			continue
		}

		stay, found := getLastStay(se.CowName)
		if !found {
			// so we dont have any last stay for that cow
			var stay Stay
			stay.Begin = se.Time
			stay.CowNr = se.CowName
			stay.Location = se.SortDst

			setLastStay(se.CowName, stay)
			continue
		}

		if se.SortDst.Id == 3 {
			registerCowInMA(se.CowName)
		} else {
			removeCowFromMA(se.CowName)
		}

		if se.SortSrc.Id != stay.Location.Id {
			// so the cow has to have magically moved between areas
			logger.Printf("%v\tStrange situation: Cow was seen leaving an area at which she wasnt staying. Cows alleged location: %v.\n", se.CowName, stay.Location.Name)
			var stay Stay
			stay.CowNr = se.CowName
			stay.Location = se.SortDst
			stay.Begin = se.Time
			setLastStay(se.CowName, stay)

			continue
		}

		if se.SortDst.Id == stay.Location.Id {
			// so we assume that last time the cow was standing at the gate but didnt go trough it
			// hence we reset the begin time of that stay
			logger.Printf("%v\tCow went trough the same gate twice. Movement: %v -> %v\n", se.CowName, se.SortSrc.Name, se.SortDst.Name)
			stay.Begin = se.Time
			continue
		}

		stay.End = se.Time
		stOut <- stay

		stay = Stay{}
		stay.CowNr = se.CowName
		stay.Location = se.SortDst
		stay.Begin = se.Time
		setLastStay(se.CowName, stay)
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
