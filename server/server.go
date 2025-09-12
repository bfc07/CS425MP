package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
)

type RemoteGrep int

type GrepResult struct {
	Hostname string
	Output   string
}

func (t *RemoteGrep) Grep(req *[]string, reply *GrepResult) error {
	// Get hostname of this machine
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}
	reply.Hostname = hostname

	// Execute grep command
	cmd := exec.Command("grep", (*req)...)
	output, err := cmd.CombinedOutput()

	reply.Output = string(output)

	if err != nil {
		// exit status 1: no match found
		// this is not considered an error
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return nil
			}
		}

		return fmt.Errorf("command execution failed: %v", err)
	}

	log.Printf("Executed command: '%s' on host: %s", *req, hostname)
	return nil
}

func main() {
	port := flag.String("port", "1234", "port number for the program to listen on")
	flag.Parse()

	// Register the RPC service
	grepServer := new(RemoteGrep)
	rpc.Register(grepServer)

	// Listen for incoming RPC connections
	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", *port, err)
	}
	defer listener.Close()

	log.Printf("listening on port %s", *port)

	// Accept and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			log.Printf("Client connected: %s", c.RemoteAddr())
			rpc.ServeConn(c)
			log.Printf("Client disconnected: %s", c.RemoteAddr())
		}(conn)
	}
}
