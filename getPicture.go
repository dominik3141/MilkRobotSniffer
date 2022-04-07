package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func takePicture(se SortEvent) {
	bodyBytes, err := json.Marshal(se)
	check(err)

	bodyReader := bytes.NewBuffer(bodyBytes)

	http.Post("172.19.60.40/api/takePicture", "application/json", bodyReader)
}
