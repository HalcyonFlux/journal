package main

import (
	"flag"
	"fmt"
	"github.com/vaitekunas/log"
	"github.com/vaitekunas/log/server"
	"os"
	"os/signal"
)

const banner = `
888                        .d8888b.
888                       d88P  Y88b
888                       Y88b.
888      .d88b.   .d88b.   "Y888b.   888d888 888  888
888     d88""88b d88P"88b     "Y88b. 888P"   888  888
888     888  888 888  888       "888 888     Y88  88P
888     Y88..88P Y88b 888 Y88b  d88P 888      Y8bd8P
88888888 "Y88P"   "Y88888  "Y8888P"  888       Y88P
                      888
                 Y8b d88P
                  "Y88P"
v0.1.0
`

func main() {

	// Remote config
	hostPtr := flag.String("host", "127.0.0.1", "Remote logger's host")
	portPtr := flag.Int("port", 4332, "Remote logger's port")
	unixSockPtr := flag.String("unix-socket", "/var/run/logsrv.sock", "Remote logger's unix socket file")
	tokenPtr := flag.String("tokens", "/opt/logsrv/tokens.db", "Remote logger's access tokens")

	// Local config
	filePtr := flag.String("filestem", "aggregate", "Log filename stem (without date and extension)")
	folderPtr := flag.String("folder", "/var/logs/logsrv", "Logserver's folder to store logs in")
	rotPtr := flag.String("rotation", "daily", "Log rotation mode: {none|daily|weekly|monthly|annually}")
	outPtr := flag.String("output", "file", "Log output mode: {file|stdout|both}")
	headPtr := flag.Bool("headers", true, "Always print headers")
	jsonPtr := flag.Bool("json", true, "Print logs encoded in json")
	compressPtr := flag.Bool("compress", true, "Compress rotated logs")

	flag.Parse()

	// Decide on rotation
	var rot int
	switch *rotPtr {
	case "daily":
		rot = log.ROT_DAILY
	case "weekly":
		rot = log.ROT_WEEKLY
	case "monthly":
		rot = log.ROT_MONTHLY
	case "annually":
		rot = log.ROT_ANNUALLY
	default:
		rot = log.ROT_NONE
	}

	// Decide on output
	var out int
	switch *outPtr {
	case "stdout":
		out = log.OUT_STDOUT
	case "both":
		out = log.OUT_FILE_AND_STDOUT
	default:
		out = log.OUT_FILE
	}

	// Complete config
	config := &server.Config{
		Banner:       banner,
		Host:         *hostPtr,
		Port:         *portPtr,
		UnixSockPath: *unixSockPtr,
		TokenPath:    *tokenPtr,

		LoggerConfig: &log.Config{
			Service:  "",
			Instance: "",
			Folder:   *folderPtr,
			Filename: *filePtr,
			Rotation: rot,
			Out:      out,
			Headers:  *headPtr,
			JSON:     *jsonPtr,
			Compress: *compressPtr,
			Columns:  []int64{}, // List of relevant columns (can be empty if default columns should be used)
		},
	}

	// Start the local logger
	logSrv, err := server.New(config)
	if err != nil {
		fmt.Printf("Could not start log server: %s\n", err.Error())
		os.Exit(1)
	}

	// Listen for sys interrupt or killswitch
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	fmt.Println("Log server is running...")
	select {
	case <-sig: // Standard os interrupt (ctrl+c)
		fmt.Println("\nReceived interrupt signal. Quitting.")
		logSrv.Quit()
	case <-logSrv.KillSwitch(): // Can be triggered via the management console
		fmt.Println("Received killswitch signal. Quitting.")
		logSrv.Quit()
	}
	fmt.Println("Log server has been shut down...")
}
