package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
)

type QueryOutput struct {
	Hostname string
	Address  string
	Output   string
	Error    error
}

type GrepResult struct {
	Hostname string
	Output   string
}

func main() {
	// hardcoded the path to the file containing IPs of all machines
	const filePath = "./sources.json"

	IPs, err := generateSliceFromJson(filePath)
	if err != nil {
		log.Fatalf("Failed to load IPs: %v", err)
	}

	if len(IPs) == 0 {
		log.Fatal("No IP addresses found in the file")
	}

	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("Usage: program <grep arguments>")
	}

	getLogAll(IPs, args)
}

func getLogAll(addresses []string, req []string) {
	// a list with length equals to the total number of machines
	results := make([]QueryOutput, len(addresses))
	var wg sync.WaitGroup

	start := time.Now()
	// create a new thread for each machine
	for i, address := range addresses {
		wg.Add(1)

		// each thread writes to its own index
		go func(index int, addr string) {
			defer wg.Done()

			result := QueryOutput{
				Address: addr,
			}

			// Connect to the RPC server
			conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
			if err != nil {
				result.Error = fmt.Errorf("connection timeout/failed: %v", err)
				result.Hostname = "Unknown" // Default when can't connect
				results[index] = result
				return
			}
			client := rpc.NewClient(conn)
			defer client.Close()

			var reply GrepResult

			// rpc call
			call := client.Go("RemoteGrep.Grep", &req, &reply, nil)
			select {
			case <-time.After(5 * time.Second):
				result.Error = fmt.Errorf("rpc call timed out")
				result.Hostname = "Unknown"
			case res := <-call.Done:
				if res.Error != nil {
					result.Error = fmt.Errorf("grep failed: %v", err)
					result.Hostname = "Unknown"
				} else {
					result.Hostname = reply.Hostname
					result.Output = reply.Output
				}
			}
			results[index] = result
		}(i, address)
	}

	wg.Wait()
	elapsed := time.Since(start)

	printFormattedResult(results, elapsed)
}

func printFormattedResult(results []QueryOutput, elapsed time.Duration) {
	successCount := 0
	failureCount := 0
	totalLines := 0

	for _, result := range results {
		fmt.Printf("\n┌─ %s (IP: %s)\n", result.Hostname, result.Address)
		fmt.Println("├" + strings.Repeat("─", 60))

		if result.Error != nil {
			fmt.Printf("│ ❌ ERROR: %v\n", result.Error)
			failureCount++
		} else if result.Output == "" {
			fmt.Println("│ ✓ No matches found")
			successCount++
		} else {
			// Count and display lines of output
			lines := strings.Split(strings.TrimSpace(result.Output), "\n")
			lineCount := len(lines)
			if lineCount > 0 && lines[0] != "" {
				totalLines += lineCount
			}

			fmt.Printf("│ ✓ Found %d matches:\n", lineCount)
			fmt.Println("│")

			// Indent each line of output for nice formatting
			for _, line := range lines {
				if line != "" {
					fmt.Printf("│   %s\n", line)
				}
			}
			successCount++
		}
		fmt.Println("└" + strings.Repeat("─", 60))
	}

	fmt.Println("\n════════════════════════════════════════════════════════════")
	fmt.Printf("SUMMARY: %d successful, %d failed out of %d machines\n",
		successCount, failureCount, len(results))
	if totalLines > 0 {
		fmt.Printf("Total matches found: %d lines\n", totalLines)
	}
	fmt.Printf("Total Latency: %v\n", elapsed)
	fmt.Println("════════════════════════════════════════════════════════════")
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
