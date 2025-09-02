package main

import (
	"fmt"
	"log"
	"net/rpc"
)

// GrepRequest defines the parameters for a grep query.
// This struct MUST match the one on the server.
type GrepRequest struct {
	Args    []string // Additional arguments for grep (e.g., "-i", "-v")
	Pattern string   // The pattern to search for
	Path    string   // The file path to search in
}

// GrepReply defines the response from a grep query.
// This struct MUST match the one on the server.
type GrepReply struct {
	Lines []string // The matching lines found by grep
}

func main() {
	// --- 1. Connect to the RPC server ---
	// Make sure the server is running and listening on this address.
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatalf("Failed to connect to RPC server: %v", err)
	}
	defer client.Close()

	// --- 2. Prepare the request ---
	// To test this, create a file named "sample.log" with some content.
	// For example:
	// INFO: Application started
	// WARNING: Deprecated function used
	// ERROR: Failed to connect to database
	// INFO: User logged in
	// error: Another failure
	request := GrepRequest{
		Path:    "/Users/bofanchen/Desktop/sample.log",
		Pattern: "GET",
		Args:    []string{"-i"}, // Example: use "-i" for case-insensitive search
	}

	// --- 3. Make the RPC call ---
	var reply GrepReply
	err = client.Call("RemoteQuery.Grep", &request, &reply)
	if err != nil {
		log.Fatalf("Error calling RemoteQuery.Grep: %v", err)
	}

	// --- 4. Process the reply ---
	fmt.Printf("Query successful. Found %d matching lines for pattern '%s' in file '%s':\n", len(reply.Lines), request.Pattern, request.Path)
	if len(reply.Lines) > 0 {
		for _, line := range reply.Lines {
			fmt.Println(line)
		}
	} else {
		fmt.Println("--- No matches found ---")
	}
}
