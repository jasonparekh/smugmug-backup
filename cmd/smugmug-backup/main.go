package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/arl/statsviz"
	log "github.com/sirupsen/logrus"
	"github.com/tommyblue/smugmug-backup"
)

var statsAddr = "localhost:6060"

// Use `-ldflags "-X main.version=someversion"` when building baker to set this value
var version = "-- unknown --"
var flagVersion = flag.Bool("version", false, "print version number")
var flagStats = flag.Bool("stats", false, fmt.Sprintf("show stats at %s", statsAddr))

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	flag, isPresent := os.LookupEnv("DEBUG")
	if isPresent && flag == "1" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)

	}
}

func main() {
	flag.Parse()
	if *flagVersion {
		fmt.Printf("Version: %s\n", version)
		return
	}

	if *flagStats {
		statsviz.RegisterDefault()
		go func() {
			log.Infof("Stats available at: http://%s/debug/statsviz/\n", statsAddr)
			log.Println(http.ListenAndServe(statsAddr, nil))
		}()
	}

	cfg, err := smugmug.ReadConf()
	if err != nil {
		log.WithError(err).Fatal("Configuration error")
	}

	wrk, err := smugmug.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("Can't initialize the package")
	}

	if err := wrk.Run(); err != nil {
		log.Fatal(err)
	}
}
