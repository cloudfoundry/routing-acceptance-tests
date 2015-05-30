package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	DEFAULT_ADDRESS                = "localhost"
	DEFAULT_START_PORT             = 3333
	DEFAULT_CONCURRENT_CONNECTIONS = 1
	DEFAULT_PORT_SPAN              = 1
	CONN_TYPE                      = "tcp"
	DEFAULT_CONNECT_TIMEOUT        = 1 * time.Second
	DEFAULT_VIRTUAL_USERS          = 1
)

var serverAddress = flag.String(
	"address",
	DEFAULT_ADDRESS,
	"The IP address of server to connect to",
)

var startPort = flag.Int(
	"startPort",
	DEFAULT_START_PORT,
	"Starting port number of IP address to connect to",
)

var numConcurrentConnections = flag.Int(
	"concurrentConnections",
	DEFAULT_CONCURRENT_CONNECTIONS,
	"Number of concurrent connections to the remote port",
)

var portSpan = flag.Int(
	"portSpan",
	DEFAULT_PORT_SPAN,
	"Number of ports starting from startPort to connect to",
)

var numVirtualUsers = flag.Int(
	"virtualUsers",
	DEFAULT_VIRTUAL_USERS,
	"Number of virtual users that connect to all the ports with concurrentConnections",
)

func main() {
	flag.Parse()

	wg := sync.WaitGroup{}
	for i := 0; i < *numVirtualUsers; i++ {
		wg.Add(1)
		go fireVirtualUsers(i, &wg)
	}
	wg.Wait()
}

func fireVirtualUsers(virtualUserId int, virtualUserWg *sync.WaitGroup) {
	defer virtualUserWg.Done()
	wg := sync.WaitGroup{}
	for i := 0; i < *portSpan; i++ {
		for j := 0; j < *numConcurrentConnections; j++ {
			wg.Add(1)
			go connect(i, j, virtualUserId, &wg)
		}
	}
	wg.Wait()
}

func log(message string) {
	now := time.Now()
	timeInSeconds := now.Unix()
	nanoSeconds := now.Nanosecond()
	fmt.Printf("%d.%d,CLIENTLOG,%s\n", timeInSeconds, nanoSeconds, message)
}

func connect(portOffset, clientId, virtualUserId int, wg *sync.WaitGroup) {
	defer wg.Done()
	port := *startPort + portOffset
	address := fmt.Sprintf("%s:%d", *serverAddress, port)
	for trycount := 0; trycount < 100; trycount++ {
		log(fmt.Sprintf("VUserID:%d Connecting to address:%s", virtualUserId, address))
		var conn net.Conn
		var err error
		for {
			conn, err = net.DialTimeout(CONN_TYPE, address, DEFAULT_CONNECT_TIMEOUT)
			if err == nil {
				log(fmt.Sprintf("VUserID:%d Successfully connected client%d to %s", virtualUserId, clientId, address))
				break
			}
			if trycount > 0 {
				log(fmt.Sprintf("Unable to connect client%d to %s...trying again", clientId, address))
			}
		}

		message := fmt.Sprintf("VUserID:%d client%d %s\n", virtualUserId, clientId, address)
		for i := 0; i < 1; i++ {
			_, err = conn.Write([]byte(message))
			if err != nil {
				log(fmt.Sprintf("VUserID:%d client%d Error writing to %s %s", virtualUserId, clientId, address, err.Error()))
			}
			time.Sleep(100 * time.Millisecond)
		}
		conn.Close()
	}
}
