package main

import (
	"bytes"
	crypto_rand "crypto/rand"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	math_rand "math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Configuration struct {
	VUsersAmount      int
	TotalTestTime     int
	TimeOut           int
	RampUpInterval    int
	VUserRampUpAmount int
	Requests          []Request
}

type Request struct {
	TYPE      string
	URL       string
	BODY      map[string]string
	ThinkTime int
}

func init() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))

}

var timeoutCount = 0

func main() {

	config := getConfig()

	vUsersAmount := config.VUsersAmount
	totalTestTime := time.Duration(config.TotalTestTime) * time.Second
	timeout := time.Duration(config.TimeOut) * time.Second
	requests := config.Requests
	//rampUpInterval := time.Duration(config.RampUpInterval) * time.Second
	//vUserRampUpAmount := config.VUserRampUpAmount

	var waitGroup sync.WaitGroup
	waitGroup.Add(vUsersAmount)

	// counts the total of requests made
	requestCount := 0

	// Saves initial timestamp to determine total testing time
	StartTime := time.Now()
	for i := 0; i < vUsersAmount; i++ {
		go func(vUserId int) {
			// counter to manage which REQUEST is being made
			requestStep := 0

			var netTransport = &http.Transport{
				TLSHandshakeTimeout: 5 * time.Second,
			}
			var client = &http.Client{
				Timeout:   timeout,
				Transport: netTransport,
			}

			for totalTestTime >= time.Since(StartTime) {
				// selectedLines saves the selected line number of the column
				selectedLines := make(map[string]int)
				if len(requests) <= requestStep {
					requestStep = 0

					// resets the list of selected line numbers here
					for k := range selectedLines {
						delete(selectedLines, k)
					}
				}

				request := requests[requestStep]

				for key, value := range request.BODY {
					// check if the parameter is surrounded in curly brackets {}
					if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {

						// delete the curly brackets
						value = strings.Replace(value, "{", "", -1)
						value = strings.Replace(value, "}", "", -1)

						// split the parameter by dots "."
						parametersConfig := strings.Split(value, ".")

						fileName := parametersConfig[0] + ".csv"
						column := parametersConfig[1]
						method := parametersConfig[2]

						fmt.Println(fileName, column, method)

						// open the file
						csvfile, err := os.Open(fileName)
						checkError("Error opening .csv file!", err)

						parameters := CSVToMap(csvfile)

						var line int
						switch method {
						case "random":
							randomInt := math_rand.Intn(len(parameters))
							line = randomInt
							selectedLines[column] = line
							fmt.Printf("random selected line for %s = %v \n parameter size %v \n", column, line, len(parameters))
						case "sameastype":
							sameascolumnName := parametersConfig[3]
							line = selectedLines[sameascolumnName]
							fmt.Printf("selected line for %s = %v \n", column, line)
						case "sequencial":
						}

						request.BODY[key] = parameters[line][column]
					}
				}

				requestBody, err := json.Marshal(request.BODY)
				checkError("Error marshaling json!", err)

				httpRequest, err := http.NewRequest(request.TYPE, request.URL, bytes.NewBuffer(requestBody))
				httpRequest.Header.Set("Content-type", "application/json")
				userAgent := fmt.Sprintf("[LoadSun's VUser ID - %v]", vUserId)
				httpRequest.Header.Set("User-Agent", userAgent)

				// Logs the error
				checkError("Error setting http request with parameters!", err)

				// The client does the request.
				resp, err := client.Do(httpRequest)

				// Logs the error
				checkError("Error doing http request!", err)

				requestCount++

				if err == nil {
					// The body of the response should be closed when it is no longer used.
					defer resp.Body.Close()

					//body, err := ioutil.ReadAll(resp.Body)
					//checkError("Error reading http response body!", err)

					log.Printf(" HTTP Response Status: %-3v %-3s  %-3v %-5v  Request step: %-4v \n", resp.StatusCode, http.StatusText(resp.StatusCode), "VUser id:", vUserId, requestStep)
				}

				requestStep++

				thinkTime := time.Duration(request.ThinkTime) * time.Second
				time.Sleep(thinkTime)
			}

			defer waitGroup.Done()

		}(i)
		//if vUserRampUpAmount == i {
		//	fmt.Printf("\n\n ADDED %v VUsers at %v \n\n", vUserRampUpAmount, time.Now())
		//	time.Sleep(rampUpInterval)
		//}
	}

	// Wait for all goroutines to finish running and sync.
	waitGroup.Wait()

	// defer function that is executed at the end of the main function informing the total number of tests and the total test time.
	defer func() {
		fmt.Printf("All tests finished in %s.\n", time.Since(StartTime))
		fmt.Printf("%-6v total requests.\n", requestCount)
		fmt.Printf("%-6v timeouts.\n", timeoutCount)
		fmt.Printf("%-6v requests excluding timeouts.\n", requestCount-timeoutCount)
	}()
}

// getConfig reads the config.json file and returns a Configuration instance.
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

// checkError does error handling.
func checkError(msg string, err error) {
	if err != nil {
		log.Print(err)
	}
	if err, ok := err.(net.Error); ok && err.Timeout() {
		timeoutCount++
	}
}

// CSVToMap takes a reader and returns an array of dictionaries, using the header row as the keys
func CSVToMap(reader io.Reader) []map[string]string {
	r := csv.NewReader(reader)
	rows := []map[string]string{}
	var header []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		checkError("error reading file", err)
		if header == nil {
			header = record
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			rows = append(rows, dict)
		}
	}
	return rows
}
