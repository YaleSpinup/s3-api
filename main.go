package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/YaleSpinup/s3-api/common"
	"github.com/YaleSpinup/s3-api/api"

	log "github.com/sirupsen/logrus"
)

var (
	// Version is the main version number
	Version = "0.0.0"

	// VersionPrerelease is a prerelease marker
	VersionPrerelease = ""

	// Buildstamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	Buildstamp = "No BuildStamp Provided"

	// Githash is the git sha of the built binary, it should be set at buildtime with ldflags
	Githash = "No Git Commit Provided"

	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("unable to get working directory")
	}
	log.Infof("Starting S3-API version %s%s (%s)", Version, VersionPrerelease, cwd)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("Unable to open config file", err)
	}

	r := bufio.NewReader(configFile)
	config, err := common.ReadConfig(r)
	if err != nil {
		log.Fatalf("Unable to read configuration from %s.  %+v", *configFileName, err)
	}

	config.Version = common.Version{
		Version:           Version,
		VersionPrerelease: VersionPrerelease,
		BuildStamp:        Buildstamp,
		GitHash:           Githash,
	}

	// Set the loglevel, info if it's unset
	switch config.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	if config.LogLevel == "debug" {
		log.Debug("Starting profiler on 127.0.0.1:6080")
		go http.ListenAndServe("127.0.0.1:6080", nil)
	}
	log.Debugf("Read config: %+v", config)

	if err := api.NewServer(config); err != nil {
		log.Fatal(err)
	}
}

func vers() {
	fmt.Printf("S3-API Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}
