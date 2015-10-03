package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/noxiouz/overlord"
	"github.com/noxiouz/overlord/version"
)

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join((*s), ";")
}

func (s *stringSlice) Set(value string) error {
	env := strings.Split(value, ";")
	(*s) = append(*s, env...)
	return nil
}

var (
	locatorFlag        string
	httpFlag           string
	slaveFlag          string
	startupTimeoutFlag time.Duration

	showVersion bool
)

func init() {
	flag.StringVar(&locatorFlag, "locator", "127.0.0.1:10053,[::1]:10053", "address of the locator")
	flag.StringVar(&httpFlag, "http", ":8080", "endpoint to serve http on")
	flag.StringVar(&slaveFlag, "slave", "", "slave path")
	flag.DurationVar(&startupTimeoutFlag, "startuptimeout", time.Minute*1, "how long we should wait for conenctin to worker")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

}

func main() {
	if showVersion {
		fmt.Printf("Overlord version: `%s`\n", version.Version)
		return
	}

	if slaveFlag == "" {
		log.Fatal("--slave must be specified")
	}

	cfg := overlord.Config{
		Slave:          slaveFlag,
		Locator:        locatorFlag,
		HTTPEndpoint:   httpFlag,
		StartUpTimeout: startupTimeoutFlag,
	}
	over, err := overlord.NewOverlord(&cfg)
	if err != nil {
		log.Fatalf("unable to start Overlord: %v", err)
	}

	if err := over.Start(); err != nil {
		log.Fatalf("Start error: %v", err)
	}
}
