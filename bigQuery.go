package main

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

type bqSortingEvent struct {
	Timestamp   time.Time
	CowName     int16
	Transponder int32
	SortSrc     int
	SortDst     int
	Gate        int
	PictureName string
}

type bqStay struct {
	CowNr    int16
	Begin    time.Time
	End      time.Time
	Location int
}

func bqInsertSE(u *bigquery.Inserter, se SortEvent, pictureName string) {
	row := bqSortingEvent{
		Timestamp:   se.Time,
		CowName:     se.CowName,
		Transponder: se.Transponder,
		SortSrc:     se.SortSrc.Id,
		SortDst:     se.SortDst.Id,
		Gate:        se.Gate.Id,
		PictureName: pictureName,
	}

	err := u.Put(context.Background(), row)
	check(err)
}

func bqInsertStay(u *bigquery.Inserter, stay Stay) {
	bqSt := bqStay{
		CowNr:    stay.CowNr,
		Begin:    stay.Begin,
		End:      stay.End,
		Location: stay.Location.Id,
	}

	err := u.Put(context.Background(), bqSt)
	check(err)
}

func bqInit() (*bigquery.Inserter, *bigquery.Inserter) {
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "rahnfarrgbr", option.WithCredentialsFile(gcpCred))
	check(err)

	dataset := client.Dataset("CowCounter")

	table1 := dataset.Table("sortings")
	u1 := table1.Inserter()

	table2 := dataset.Table("stays")
	u2 := table2.Inserter()
	return u1, u2
}

// bqInitDb creates a new dataset and a new table with a schema infered from
// the corresponding struct
func bqInitTables(client *bigquery.Client) {
	ctx := context.Background()

	dataset := client.Dataset("CowCounter")
	err := dataset.Create(ctx, &bigquery.DatasetMetadata{Location: "europe-west3"})
	check(err)

	// SORTINGS
	table := dataset.Table("sortings")

	// construct schema
	schema, err := bigquery.InferSchema(bqSortingEvent{})
	check(err)

	// create table with schema
	err = table.Create(ctx, &bigquery.TableMetadata{Schema: schema})
	check(err)

	// STAYS
	table = dataset.Table("stays")

	// construct schema
	schema, err = bigquery.InferSchema(bqStay{})
	check(err)

	// create table with schema
	err = table.Create(ctx, &bigquery.TableMetadata{Schema: schema})
	check(err)
}
