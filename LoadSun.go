package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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
	TYPE   string
	URL    string
	BODIES []map[string]string
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

	// conta o total de requests
	requestCount := 0

	// Salva o timestamp incial para determinar o tempo total de teste
	StartTime := time.Now()
	for i := 0; i < vUsersAmount; i++ {
		go func(i int) {
			// contador para saber em qual REQUEST esta
			requestStep := 0

			client := &http.Client{
				Timeout: timeout,
			}

			for totalTestTime >= time.Since(StartTime) {
				if len(requests) <= requestStep {
					requestStep = 0

					// resetar sameastype aqui
				}

				request := requests[requestStep]

				// declara um httpRequest apenas para mudar dentro do if, dependendo se tem BODY ou não,
				// os parametros aqui declarados não são utilizados.
				httpRequest, err := http.NewRequest("", "", nil)

				// verifica se a request tem alguma coisa no array de bodies para criar um httpRequest COM ou SEM body.
				bodiesSize := len(requests[requestStep].BODIES)
				if bodiesSize > 0 {
					selectedBody := requests[requestStep].BODIES[rand.Intn(bodiesSize)]
					requestBody, err := json.Marshal(selectedBody)

					httpRequest, err = http.NewRequest(request.TYPE, request.URL, bytes.NewBuffer(requestBody))
					httpRequest.Header.Set("Content-type", "application/json")

					// Se der erro da um log no erro
					if err != nil {
						log.Fatalln(err)
					}
				} else {
					httpRequest, err = http.NewRequest(request.TYPE, request.URL, nil)
					httpRequest.Header.Set("Content-type", "application/json")

					// Se der erro da um log no erro
					if err != nil {
						log.Fatalln(err)
					}
				}

				// o cliente faz a requisição.
				resp, err := client.Do(httpRequest)

				// Se der erro da um log no erro.
				if err != nil {
					log.Fatalln(err)
				}

				requestCount++

				// deve-se fechar o corpo da resposta quando não se for utiliza-la mais.
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

	// Espera todas as goroutines terminarem de executar e sincroniza.
	waitGroup.Wait()

	// função defer q é executada no final da main informando o numero total de testes e o tempo de teste.
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
