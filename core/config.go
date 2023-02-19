package core

import (
	"encoding/json"
	"fmt"
	"os"
)

/*
	Configuration
*/

var GodbFileName string

var ListenAddress string   // address redirector process should listen on
var ListenPort int         // port redirector process should listen on
var ExternalAddress string // NAT/external address where users interface
var ExternalPort int       // should be optional
var ExternalProto string   // http or https
var RedirectorName string  // defaults to 'go2'
var PruneInterval string   // number of seconds to wait between link burnings
var NewListBehavior string // default behavior for new lists
var LevDistRatio float64   // ratio of lev distance to term length
var LinkLogNewKeywords bool
var LinkLogCapacity int
var LogFile string
var FailoverPeer string
var FailoverLocal string

type Config struct {
	LocalListenAddress string  `json:"local_listen_address"`
	LocalListenPort    int     `json:"local_listen_port"`
	ExternalAddress    string  `json:"external_address"`
	ExternalPort       int     `json:"external_port"`
	ExternalProto      string  `json:"external_proto"`
	GodbFilename       string  `json:"godb_filename"`
	RedirectorName     string  `json:"redirector_name"`
	PruneInterval      string  `json:"prune_interval"`
	NewListBehavior    string  `json:"new_list_behavior"`
	LinkLogNewKeywords bool    `json:"link_log_new_keywords"`
	LinkLogCapacity    int     `json:"link_log_capacity"`
	LevDistRatio       float64 `json:"levenshtein_distance_ratio"`
	LogFile            string  `json:"log_file"`
	FailoverPeer       string  `json:"failover_peer"`
	FailoverLocal      string  `json:"failover_local"`
}

// RenderConfig parses config.json off the disk and returns a Config struct with an err value.
func RenderConfig(file string) (Config, error) {
	var parsed Config
	cfgFile, err := os.Open(file)

	if err != nil {
		err = fmt.Errorf("error loading %s: %s (run install script)", file, err)
		return parsed, err
	}
	defer cfgFile.Close()

	parser := json.NewDecoder(cfgFile)
	err = parser.Decode(&parsed)

	// If it got past decoding, now we do validation on the actual values here.
	if parsed.ExternalProto == "" {
		err = fmt.Errorf("external_proto must be 'http' or 'https' in config file")
	}

	return parsed, err
}
