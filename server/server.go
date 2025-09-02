package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
)

type RemoteQuery int

// GrepRequest defines the parameters for a grep query.
type GrepRequest struct {
	Args    []string // Additional arguments for grep (e.g., "-i", "-v")
	Pattern string   // The pattern to search for
	Path    string   // The file path to search in
}

// GrepReply defines the response from a grep query.
type GrepReply struct {
	Lines []string // The matching lines found by grep
}

// Grep is the RPC method that executes the grep command on the server.
func (t *RemoteQuery) Grep(req *GrepRequest, reply *GrepReply) error {
	// --- 1. Prepare the command and arguments ---
	// Combine the user-provided args with the pattern.
	// Example: if req.Args is ["-i"] and req.Pattern is "error", this becomes ["-i", "error"]
	args := append(req.Args, req.Pattern)
	cmd := exec.Command("grep", args...)

	// --- 2. Set up input streaming ---
	file, err := os.Open(req.Path)
	if err != nil {
		// Return an error instead of killing the server with log.Fatalf
		log.Printf("Failed to open file '%s': %v", req.Path, err)
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Connect the file directly to grep's standard input for efficient streaming.
	cmd.Stdin = file

	// --- 3. Set up output streaming ---
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to get stdout pipe: %v", err)
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// --- 4. Start the command ---
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start command: %v", err)
		return fmt.Errorf("failed to start grep command: %w", err)
	}

	// --- 5. Process the output and populate the reply ---
	// Initialize the slice to ensure it's not nil if no lines are found.
	reply.Lines = []string{}
	scanner := bufio.NewScanner(stdoutPipe)

	// Read each line of output from grep as it comes in.
	for scanner.Scan() {
		// CRITICAL CHANGE: Append the found line to the reply struct.
		reply.Lines = append(reply.Lines, scanner.Text())
	}

	// Check for any errors during scanning (e.g., from a broken pipe).
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from stdout pipe: %v", err)
		return fmt.Errorf("error reading grep output: %w", err)
	}

	// --- 6. Finalize and handle exit codes ---
	// Wait for the command to finish.
	if err := cmd.Wait(); err != nil {
		// grep exits with status 1 if no lines are found. This is normal and not a "real" error.
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// This is a successful execution, just with zero results.
			// The reply.Lines is already an empty slice, so we're done.
			return nil
		}
		// For any other error (e.g., exit code 2, command killed), it's a real problem.
		log.Printf("Command finished with unexpected error: %v", err)
		return fmt.Errorf("grep command failed: %w", err)
	}

	// If everything went perfectly, return nil to indicate success.
	return nil
}

func main() {
	server := new(RemoteQuery)
	rpc.Register(server)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	http.Serve(l, nil)
}
