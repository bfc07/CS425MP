package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a string argument.")
		return
	}

	addressFile := "sources.json"
	IPs, err := generateSliceFromJson(addressFile)
	if err != nil {
		fmt.Println("Addresses to grep from not found")
		return
	}

	fmt.Println(IPs)

	// create a wait group
	var wg sync.WaitGroup

	for _, ip := range IPs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			getLog(addr)
		}(ip)
	}

	wg.Wait()
}

func getLog(address string) {
	// Connect to the RPC server
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	request := strings.Join(os.Args[1:], " ")

	// Prepare the variable to hold the response
	var reply string

	// Make the RPC call to the server
	err = client.Call("RemoteGrep.Grep", request, &reply)
	if err != nil {
		log.Fatalf("RemoteGrep error: %v", err)
	}

	// Print the server's response
	fmt.Printf(reply)
}

func generateSliceFromJson(filePath string) ([]string, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var list []string
	err = json.Unmarshal(file, &list)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return list, nil
}
