package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os/exec"
)

type RemoteGrep int

func (t *RemoteGrep) Grep(req *string, reply *string) error {
	cmd := exec.Command("sh", "-c", *req)

	output, err := cmd.CombinedOutput()
	if err != nil {
		*reply = string(output)
		return fmt.Errorf("command execution failed: %v, output: %s", err, output)
	}

	*reply = string(output)
	log.Printf("Executed command: '%s'.", *req)

	return nil
}

func main() {
	// Register the RPC service.
	grepServer := new(RemoteGrep)
	rpc.Register(grepServer)

	// Listen for incoming RPC connections
	port := ":1234"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}
	defer listener.Close()

	log.Printf("RPC server for RemoteGrep is listening on port %s", port)

	// Accept and handle incoming connections.
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}
