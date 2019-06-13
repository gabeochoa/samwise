package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	var files []string

	root := "./data"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, "venv") {
			return nil
		}
		if strings.Contains(path, ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	// first we have to create the folder

	folderURL := "http://localhost:8080/api/v1/folders/pokemon"
	req, err := http.NewRequest("POST", folderURL, bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
		return
	}
	// fmt.Println("response Status:", resp.Status)
	// fmt.Println("response Headers:", resp.Header)
	// _, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println("response Body:", string(body))

	resp.Body.Close()

	var wg sync.WaitGroup
	wg.Add(len(files))

	for _, file := range files {

		func(file string) {

			defer wg.Done()
			// fmt.Println(file)
			// Open our jsonFile
			jsonFile, err := ioutil.ReadFile(file)
			if err != nil {
				fmt.Println(err)
			}

			poke := file[5 : len(file)-5]
			url := fmt.Sprintf("http://localhost:8080/api/v1/pokemon/%s", poke)
			// fmt.Println(url)

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonFile))
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			// fmt.Println("response Status:", resp.Status)
			// fmt.Println("response Headers:", resp.Header)
			// body, _ := ioutil.ReadAll(resp.Body)
			// fmt.Println("response Body:", string(body))
			// fmt.Println(string(body))
			resp.Body.Close()

			return
		}(file)
		// break
	}

	wg.Wait()
}
