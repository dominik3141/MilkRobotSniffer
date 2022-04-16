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
	// PictureName string
}

func bqInsertSE(u *bigquery.Inserter, se SortEvent, pictureName string) {
	row := bqSortingEvent{
		Timestamp:   se.Time,
		CowName:     se.CowName,
		Transponder: se.Transponder,
		SortSrc:     se.SortSrc.Id,
		SortDst:     se.SortDst.Id,
		Gate:        se.Gate.Id,
		// PictureName: pictureName,
	}

	err := u.Put(context.Background(), row)
	check(err)
}

func bqInit() *bigquery.Inserter {
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "rahnfarrgbr", option.WithCredentialsFile(gcpCred))
	check(err)

	dataset := client.Dataset("CowCounter")
	table := dataset.Table("sortings")

	u := table.Inserter()
	return u
}

// bqInitDb creates a new dataset and a new table with a schema infered from
// the bqSortingEvent struct
func bqInitDb(client *bigquery.Client) {
	ctx := context.Background()

	dataset := client.Dataset("CowCounter")
	err := dataset.Create(ctx, &bigquery.DatasetMetadata{Location: "europe-west3"})
	check(err)
	table := dataset.Table("sortings")

	// construct schema
	schema, err := bigquery.InferSchema(bqSortingEvent{})
	check(err)

	// create table with schema
	err = table.Create(ctx, &bigquery.TableMetadata{Schema: schema})
	check(err)
}
