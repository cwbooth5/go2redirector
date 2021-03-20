package core

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

/*
This is an active/standby model with the standby being a hot standby.

Heartbeat mechanism: simple ping at a given time interval. We set a threshold
for number of heartbeats we can miss before failing over/assuming an active role.

Active and Standby states:
  - The standby does not handle incoming requests until it is active. This prevents
    a confusing active-active situation. Being down on the HTTP client-facing side also
	has the benefit of acting as the healthcheck metric for the upstream load balancer,
	whatever that may be.
  - When the redirector comes up, it determines it is active if no HA peer is listed in the config.
  - If there is an HA peer listed in the config, the two systems need to determine an active system.

  The protocol for determining active and standby goes like this.

	  1. Each redirector generates a random number between 0 and a million.
	  2. Each system opens a TCP connection to the peer, sharing their number.
	  3. The system with the higher number becomes active. The lower number system becomes standby.

Failover mechanism: The standby pings the active at a time interval.

Data sharing: We keep track of the newer link database with a simple epoch in the db itself.

*/

// higher number wins and becomes active
var ActiveStandbySeed = generateSeed()

// enum/label determining if this is the active or standby
// We start active until we lose a dice roll.
var IsActiveRedirector = true

// function to maintain the TCP connection to the peer

// function to load in the incoming link database updates

// Generate a random (enough) number from a time seed.
// This is used to figure out if we are active or standby on startup.
func generateSeed() int {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return r.Intn(1000000)
}

func (d LinkDatabase) ExportNetwork() error {
	file, err := json.Marshal(d)
	if err != nil {
		LogError.Println("JSON marshal error:", err)
		return err
	}
	serviceAddress, err := net.ResolveTCPAddr("tcp4", FailoverPeer)
	if err != nil {
		LogError.Fatalf("couldn't convert IP:port tuple to a valid service address: %s", err)
	}
	conn, err := net.DialTCP("tcp4", nil, serviceAddress)
	if err != nil {
		LogError.Printf("Standby peer status: %s\n", err)
		return err
	}

	LogInfo.Println("Standby peer is up, sending update...")
	if _, err := conn.Write(file); err != nil {
		LogDebug.Fatal(err)
	}
	conn.Write(file)
	conn.Close()
	return err
}

// This sends an entire link database out on the wire.
// It needs to be improved to only send incremental updates.
func SendUpdates(ldb *LinkDatabase) {
	for {
		LogDebug.Println("send update called...")
		time.Sleep(3 * time.Second)
		ldb.ExportNetwork()
	}
}

// This handles incoming updates from the active redirector peer.
func RunFailoverMonitor(updates chan *LinkDatabase) {
	serviceAddress, err := net.ResolveTCPAddr("tcp4", FailoverLocal)
	if err != nil {
		LogError.Fatalf("couldn't convert IP:port tuple to a valid service address: %s", err)
	}

	listener, err := net.ListenTCP("tcp", serviceAddress)
	if err != nil {
		LogError.Fatalf("couldn't open listening TCP socket at %s\n", FailoverLocal)
	}
	LogInfo.Printf("failover monitor started, listening on: %s\n", FailoverLocal)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// Two cases here.
		// 1. The data arriving is a diceroll from a peer coming up. (we are active)
		// 2. The data arriving is a DB we need to load into memory. (we are standby)
		data, err := ioutil.ReadAll(conn)
		result := strings.TrimSpace(fmt.Sprintf("%s", data))
		if strings.HasPrefix(result, "diceroll") {
			// case 1
			// Send back a high number, impossible for the peer to beat, forcing them standby.
			LogDebug.Println("A peer just came online, letting them know we are active")
			conn.Write([]byte("diceroll:9999999"))
			conn.Close()
		} else {
			// case 2
			// We try to marshal the incoming data into JSON, if so, it's a DB update.
			// We send back a message about how things went, for informational purposes.
			var tempdb LinkDatabase
			reader := strings.NewReader(result)
			dec := json.NewDecoder(reader)
			if err := dec.Decode(&tempdb); err == io.EOF {
				continue
			} else if err != nil {
				LogError.Fatal(err)
			}
			// send a positive acknowledgement to the standby
			conn.Write([]byte("SUCCESS"))
			conn.Close()
			updated_database := &tempdb
			updates <- updated_database
		}
	}
}

func Synchronize() {
	serviceAddress, err := net.ResolveTCPAddr("tcp4", FailoverPeer)
	if err != nil {
		LogError.Fatalf("couldn't convert IP:port tuple to a valid service address: %s", err)
	}
	conn, err := net.DialTCP("tcp4", nil, serviceAddress)

	if err != nil {
		LogInfo.Printf("Failed to connect to peer: %v", err)
		LogInfo.Println("Failover peer unreachable, assuming active role")
		LogInfo.Printf("Initial sync complete. active == %v\n", IsActiveRedirector)
		return
	}
	defer conn.Close()
	LogDebug.Println("Peer was found, exchanging sync information...")
	if _, err := conn.Write([]byte(fmt.Sprintf("diceroll:%d", ActiveStandbySeed))); err != nil {
		log.Fatal(err)
	}
	conn.CloseWrite()

	r, err := ioutil.ReadAll(conn)
	// result should be a diceroll from the other side. See who's higher.
	result := strings.TrimSpace(string(r))
	if strings.HasPrefix(result, "diceroll") {
		s := strings.Split(result, ":")
		if len(s) > 1 {
			theirRoll, _ := strconv.Atoi(s[1])
			if ActiveStandbySeed < theirRoll {
				LogInfo.Printf("Peer: %d, Us: %d (we are standby)\n", theirRoll, ActiveStandbySeed)
				IsActiveRedirector = false
			} else {
				LogInfo.Printf("Peer: %d, Us: %d (we are active)\n", theirRoll, ActiveStandbySeed)
			}
		} else {
			LogError.Println("garbage input from the peer")
		}
	} else {
		LogError.Println("TCP connection is up, but the diceroll prefix wasn't found.")
	}
	LogInfo.Printf("Initial sync complete. active == %v\n", IsActiveRedirector)
}
