package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Configuration struct {
	VUsersAmount  int
	ThinkTime     int
	TotalTestTime int
	TimeOut       int
	Requests      []Request
}

type Request struct {
	TYPE string
	URL  string
	BODY map[string]string
}

func main() {

	config := getConfig()

	vUsersAmount := config.VUsersAmount
	thinkTime := time.Duration(config.ThinkTime) * time.Second
	totalTestTime := time.Duration(config.TotalTestTime) * time.Second
	timeout := time.Duration(config.TimeOut) * time.Second
	requests := config.Requests

	var waitGroup sync.WaitGroup
	waitGroup.Add(vUsersAmount)

	requestCount := 0 // conta o total de requests

	StartTime := time.Now() // Salva o timestamp incial para determinar o tempo total de teste
	for i := 0; i < vUsersAmount; i++ {
		go func(i int) {
			requestStep := 0

			client := &http.Client{
				Timeout: timeout,
			}

			for totalTestTime >= time.Since(StartTime) {
				if len(requests) <= requestStep {
					requestStep = 0
				}

				request := requests[requestStep]

				requestBody, err := json.Marshal(request.BODY)

				httpRequest, err := http.NewRequest(request.TYPE, request.URL, bytes.NewBuffer(requestBody))
				httpRequest.Header.Set("Content-type", "application/json")

				// Se der erro da um log no erro
				if err != nil {
					log.Fatalln(err)
				}

				resp, err := client.Do(httpRequest)

				requestCount++

				// Se der erro da um log no erro
				if err != nil {
					log.Fatalln(err)
				}

				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				log.Printf("%s VUser id: %v\n\n", string(body), i)
				requestStep++
				time.Sleep(thinkTime)
			}
			defer waitGroup.Done()

		}(i)
	}

	waitGroup.Wait() // Espera todas as goroutines terminarem de executar e sincroniza

	defer func() {
		fmt.Printf("All tests finished in %s.\n", time.Since(StartTime))
		fmt.Printf("%v total requests.\n", requestCount)
	}()
}

func getConfig() Configuration {
	file, _ := os.Open("config.json")
	defer file.Close()
	config := Configuration{}
	err := json.NewDecoder(file).Decode(&config)
	if err != nil {
		fmt.Println("error:", err)
	}
	return config
}
