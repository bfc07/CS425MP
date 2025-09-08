package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
)

const timeout = 5 * time.Second    // network timeout
const timeLimit = 20 * time.Second // RPC deadline

type nodeResult struct {
	addr      string
	output    string
	err       error // includes grep exit statuses surfaced by the server
	reachable bool  // network-level reachability (dial/call succeeded)
}

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

	request := strings.Join(os.Args[1:], " ")

	resultsCh := make(chan nodeResult, len(IPs))
	var wg sync.WaitGroup

	for _, ip := range IPs {
		addr := strings.TrimSpace(ip)
		if addr == "" {
			continue
		}
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			resultsCh <- getLogResult(a, request)
		}(addr)
	}

	wg.Wait()
	close(resultsCh)

	// Print per-node outputs and compute summary
	total := len(IPs)
	completed := 0
	var unreachable []string

	for res := range resultsCh {
		fmt.Printf("===== %s =====\n", res.addr)
		if !res.reachable {
			fmt.Printf("[warn] unreachable: %v\n\n", res.err)
			unreachable = append(unreachable, res.addr)
			continue
		}
		// Node reached; count as completed regardless of grep exit code
		completed++

		// Show grep output (may be empty) and any server-reported error
		if res.output != "" {
			fmt.Print(res.output)
			if !strings.HasSuffix(res.output, "\n") {
				fmt.Println()
			}
		}
		if res.err != nil {
			// Grep exit != 0 or other server-side error
			fmt.Printf("[warn] %s: %v\n", res.addr, res.err)
		}
		fmt.Println()
	}

	// Summary footer
	if len(unreachable) == 0 {
		fmt.Printf("Completed: %d/%d nodes. All reachable.\n", completed, total)
	} else {
		fmt.Printf("Completed: %d/%d nodes. Unreachable: %s\n",
			completed, total, strings.Join(unreachable, ", "))
	}
}

func getLogResult(address, request string) nodeResult {
	// Dial with timeout so slow/down nodes don't block others.
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", address)
	if err != nil {
		return nodeResult{addr: address, reachable: false, err: fmt.Errorf("dial: %w", err)}
	}
	defer conn.Close()

	// Set an overall deadline covering the RPC call
	_ = conn.SetDeadline(time.Now().Add(timeLimit))

	client := rpc.NewClient(conn)
	defer client.Close()

	var reply string
	if err := client.Call("RemoteGrep.Grep", request, &reply); err != nil {
		// Reached the node (reachable = true), but RPC/grep failed
		return nodeResult{addr: address, reachable: true, output: reply, err: err}
	}

	// Clear deadline to avoid spurious errors on close
	_ = conn.SetDeadline(time.Time{})

	return nodeResult{addr: address, reachable: true, output: reply, err: nil}
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
