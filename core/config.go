package core

import (
	"encoding/json"
	"os"
)

/*
	Configuration
*/

var GodbFileName string

var ListenAddress string
var ListenPort int
var ExternalAddress string
var ExternalPort int
var RedirectorName string
var PruneInterval string
var NewListBehavior string
var LevDistRatio float64
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
	defer cfgFile.Close()
	if err == nil {
		parser := json.NewDecoder(cfgFile)
		err = parser.Decode(&parsed)
	}
	return parsed, err
}
