package main

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

var ctx context.Context
var bkt *storage.BucketHandle

func initTakePicture() *storage.Client {
	// init connection
	// create context
	ctx = context.Background()

	// create client
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(gcpCred))
	check(err)

	return client
}

func takePicture(se SortEvent, client *storage.Client) (string, error) {
	var camIp string
	switch se.Gate.Id {
	case 1:
		camIp = "172.19.60.75"
	case 2:
		camIp = "172.19.60.76"
	case 3:
		camIp = "172.19.60.75"
	case 4:
		camIp = "172.19.60.71"
	case 5:
		camIp = "172.19.60.72"
	case 6:
		camIp = "172.19.60.73"
	case 7:
		camIp = "172.19.60.74"
	default:
		return "", errors.New("Unknown camIp.")
	}

	resp, err := http.Get("http://rahnfarr:rahnfarr@" + camIp + "/jpg/image.jpg?size=3")
	if err != nil {
		return "", err
	}

	picture := make([]byte, resp.ContentLength)
	n, err := io.ReadFull(resp.Body, picture)
	if err != nil {
		return "", err
	}
	if n != int(resp.ContentLength) {
		fmt.Println("ERROR: n!=resp.ContentLength")
		return "", err
	}

	hash := sha1.Sum(picture)
	fmt.Printf("Attempting to save file with hash: %x\n", hash)
	objName := fmt.Sprintf("%x", hash)

	// get bucket
	bkt = client.Bucket("selectionpictures")

	// create an object handle
	obj := bkt.Object(objName)

	// create an io.Writer for the object
	w := obj.NewWriter(ctx)

	// write to the object
	n, err = w.Write(picture)
	fmt.Printf("Wrote %v bytes to bucket\n", n)

	// close object
	err = w.Close()
	check(err)

	return objName, nil
}
