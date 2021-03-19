package core

import (
	"context"
	"fmt"
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

// function to export outgoing link database updates
// func (d *core.LinkDataBase) SendUpdate() error {
// 	var err error
// 	file, err := json.Marshal(*d)
// 	if err != nil {
// 		LogError.Println("JSON marshal error:", err)
// 		return err
// 	}
// }

// Generate a random (enough) number from a time seed.
// This is used to figure out if we are active or standby on startup.
func generateSeed() int {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return r.Intn(1000000)
}

func FailoverMonitor() {

}

func Synchronize() {
	// tcpAddr, err := net.ResolveTCPAddr("tcp4", core.FailoverPeer)
	// if err != nil {
	// 	log.Fatal("bad address:port in config file")
	// }
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", FailoverPeer)
	if err != nil {
		log.Printf("Failed to connect to peer: %v", err)
		log.Println("Failover peer unreachable, assuming active role")
		return
	}
	defer conn.Close()
	log.Println("Peer was found, exchanging sync information...")
	if _, err := conn.Write([]byte(fmt.Sprintf("diceroll:%d", generateSeed()))); err != nil {
		log.Fatal(err)
	}

	r, err := ioutil.ReadAll(conn)
	// result should be a diceroll from the other side. See who's higher.
	result := strings.TrimSpace(string(r))
	fmt.Println(result)
	if strings.HasPrefix(result, "diceroll") {
		s := strings.Split(result, ":")
		if len(s) > 1 {
			theirRoll, _ := strconv.Atoi(s[1])
			if ActiveStandbySeed < theirRoll {
				fmt.Printf("Peer: %d, Us: %d (we are standby)\n", theirRoll, ActiveStandbySeed)
				IsActiveRedirector = false
			} else {
				fmt.Printf("Peer: %d, Us: %d (we are active)\n", theirRoll, ActiveStandbySeed)
			}
		} else {
			fmt.Println("garbage input from the peer")
		}
	} else {
		fmt.Println("TCP connection is up, but the diceroll prefix wasn't found.")
	}
}
