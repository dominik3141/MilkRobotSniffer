package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type GCPObjId struct {
	ObjName string `json:"ObjName"`
}

func takePicture(se SortEvent) string {
	if !(se.Gate.Id == 3 || se.Gate.Id == 2) {
		return ""
	}

	bodyBytes, err := json.Marshal(se)
	if err != nil && err != io.EOF {
		fmt.Println("ERROR:", err)
		return ""
	}

	bodyReader := bytes.NewBuffer(bodyBytes)

	resp, err := http.Post("http://172.19.60.40:1208/api/takePicture", "application/json", bodyReader)
	if err != nil && err != io.EOF {
		fmt.Println("ERROR:", err)
		return ""
	} else {
		// defer resp.Body.Close()
		_, err = io.Copy(os.Stdout, resp.Body)
		if err != nil && err != io.EOF {
			fmt.Println("ERROR:", err)
			return ""
		}
		fmt.Printf("\n")
	}

	var objName GCPObjId
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	fmt.Printf("Read %v bytes from body\n", n)
	if err != nil && err != io.EOF {
		fmt.Println("ERROR:", err)
		return ""
	}
	err = json.Unmarshal(buf, &objName)
	if err != nil && err != io.EOF {
		fmt.Println("ERROR:", err)
		return ""
	}

	return objName.ObjName
}
