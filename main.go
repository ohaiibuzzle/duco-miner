// Program to mine Duino-Coin.
package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var username string = ""                // User to mine to.
var diff string = ""                    // Possible safe values: MEDIUM, LOW, NET.
var x int = 1                           // Goroutines count.
var addr string = "103.253.43.216:3674" // Pool's IP:Pool's port for v3.0 .
var miningKey string = " "              // Mining key for v3.0.

// Shares
var accepted int = 0
var rejected int = 0

var start_time time.Time = time.Now()
var thread_hases []int

type PoolInfo struct {
	Client  string `json:"client"`
	IP      string `json:"ip"`
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Region  string `json:"region"`
	Server  string `json:"server"`
	Success bool   `json:"success"`
}

func work(threadID int) {
	conn, _ := net.Dial("tcp", addr)
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	log.Println("Server is on version: " + string(buffer))

	if err != nil {
		log.Println("Servers might be down or a routine may have restarted, quitting routine.")
		return
	}

	for {
		// Requesting a job.
		job_request := "JOB," + username + "," + diff + "," + miningKey
		// Send enconded as utf-8 string.
		_, err = conn.Write([]byte(job_request))

		if err != nil {
			log.Println(err)
			log.Fatal("Error requesting job.")
		}

		// Making a buffer for the job.
		buffer := make([]byte, 2048)
		_, err = conn.Read(buffer) // Getting the jobs.
		// log.Println("Received job: " + string(buffer)) // Debugging purposes.

		if err != nil {
			log.Println(err)
			log.Fatal("Error receiving job.")
		}

		buffer = bytes.Trim(buffer, "\x00")
		job := strings.Split(strings.TrimSpace(string(buffer)), ",") // Parsing the job.
		hash := job[0]
		goal := job[1]

		// Removes null bytes from job then converts it to an int.
		diff, _ := strconv.Atoi(job[2])

		for i := 0; i <= diff*100; i++ {
			thread_hases[threadID]++
			h := sha1.New()
			h.Write([]byte(hash + strconv.Itoa(i))) // Hash
			nh := hex.EncodeToString(h.Sum(nil))
			if nh == goal {
				// Sends the result of hash algorithm to the pool.
				_, err = conn.Write([]byte(strconv.Itoa(i)))

				if err != nil {
					log.Println("Error writing hash result")
					log.Fatal(err)
					break
				}

				feedback_buffer := make([]byte, 1024)
				_, err = conn.Read(feedback_buffer) // Reads response.

				if err != nil {
					log.Println("Error receiving feedback")
					log.Fatal(err)
				}

				feedback_buffer = bytes.Trim(feedback_buffer, "\x00")
				feedback := (strings.TrimSpace(string(feedback_buffer)))

				if feedback == "GOOD" || feedback == "BLOCK" {
					accepted++
				} else if feedback == "BAD" {
					rejected++
				} else if feedback == "INVU" {
					log.Fatal("Invalid username received in feedback")
				}
			}
		}
	}
}

func main() {
	argsWithoutProg := os.Args[1:]

	log.Println("Go miner started... ")

	if len(argsWithoutProg) == 0 {
		// Read from env variables.
		username = os.Getenv("MINER_USERNAME")
		x, _ = strconv.Atoi(os.Getenv("MINER_THREADS"))
		diff = os.Getenv("MINER_DIFFICULTY")
		miningKey = os.Getenv("MINER_KEY")
	} else if len(argsWithoutProg) > 0 {
		// Passing command line interface's arguments.
		username = os.Args[1]
		x, _ = strconv.Atoi(os.Args[2])
		diff = os.Args[3]
		miningKey = os.Args[4]
	}

	if username == "" || diff == "" || miningKey == "" {
		binaryName := os.Args[0]
		log.Println("Usage: " + binaryName + " <username> <threads> <difficulty> <miningKey>")
		log.Fatal("Invalid arguments, please check your arguments.")
	}

	string_count := strconv.Itoa(x)

	log.Println("Username: " + username)
	log.Println("Goroutines count: " + string_count)
	log.Println("Difficulty: " + diff)

	// Get pool info from https://server.duinocoin.com/getPool
	req, err := http.NewRequest("GET", "https://server.duinocoin.com/getPool", nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error getting pool info, using default pool.")
	} else {
		defer resp.Body.Close()
		var poolInfo PoolInfo
		err := json.NewDecoder(resp.Body).Decode(&poolInfo)
		if err != nil {
			log.Println("Error decoding pool info, using default pool.")
		} else {
			addr = poolInfo.IP + ":" + strconv.Itoa(poolInfo.Port)
			log.Println("Using pool: " + addr)
		}
	}

	thread_hases = make([]int, x)

	for i := 0; i < x; i++ {
		go work(i)
		time.Sleep(1 * time.Second)
	}

	for {
		log.Printf("Accepted shares: %d Rejected shares: %d\n", accepted, rejected)
		total_hashes := 0
		for i := 0; i < x; i++ {
			total_hashes += thread_hases[i]
			thread_hases[i] = 0
		}
		log.Printf("Hashrate: %f MH/s\n", float64(total_hashes)/time.Since(start_time).Seconds()/1000000)
		start_time = time.Now()
		time.Sleep(10 * time.Second)
	}
}
