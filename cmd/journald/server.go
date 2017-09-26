package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/vaitekunas/journal"
	"github.com/vaitekunas/journal/server"
)

// StartServer starts the journald server
func StartServer(srv *flag.FlagSet) {

	// Remote config
	hostPtr := srv.String("host", "127.0.0.1", "Remote logger's host")
	portPtr := srv.Int("port", 4332, "Remote logger's port")
	unixSockPtr := srv.String("unix-socket", "/var/run/journald.sock", "Remote logger's unix socket file")
	tokenPtr := srv.String("tokens", "/opt/journald/tokens.db", "Remote logger's access tokens")
	statsPtr := srv.String("stats", "/opt/journald/stats.db", "Remote logger's statistics")

	// Local config
	filePtr := srv.String("filestem", "aggregate", "Log filename stem (without date and extension)")
	folderPtr := srv.String("folder", "/var/logs/journald", "Logserver's folder to store logs in")
	rotPtr := srv.String("rotation", "daily", "Log rotation mode: {none|daily|weekly|monthly|annually}")
	outPtr := srv.String("output", "file", "Log output mode: {file|stdout|both}")
	headPtr := srv.Bool("headers", true, "Always print headers")
	jsonPtr := srv.Bool("json", true, "Print logs encoded in json")
	compressPtr := srv.Bool("compress", true, "Compress rotated logs")

	srv.Parse(os.Args[2:])

	// Decide on rotation
	var rot int
	switch *rotPtr {
	case "daily":
		rot = journal.ROT_DAILY
	case "weekly":
		rot = journal.ROT_WEEKLY
	case "monthly":
		rot = journal.ROT_MONTHLY
	case "annually":
		rot = journal.ROT_ANNUALLY
	default:
		rot = journal.ROT_NONE
	}

	// Decide on output
	var out int
	switch *outPtr {
	case "stdout":
		out = journal.OUT_STDOUT
	case "both":
		out = journal.OUT_FILE_AND_STDOUT
	default:
		out = journal.OUT_FILE
	}

	// Complete config
	config := &server.Config{
		Host:         *hostPtr,
		Port:         *portPtr,
		UnixSockPath: *unixSockPtr,
		TokenPath:    *tokenPtr,
		StatsPath:    *statsPtr,

		LoggerConfig: &journal.Config{
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

	// Management console
	manager := server.NewConsole()

	// Start the local logger
	journald, err := server.New(config, manager)
	if err != nil {
		fmt.Printf("Could not start log server: %s\n", err.Error())
		os.Exit(1)
	}

	// Listen for sys interrupt or killswitch
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	fmt.Println(banner)
	fmt.Printf("journald is running...\n\n")
	select {
	case <-sig: // Standard os interrupt (ctrl+c)
		fmt.Println("\nReceived interrupt signal. Quitting.")
		journald.Quit()
	case <-journald.KillSwitch(): // Can be triggered via the management console
		fmt.Println("Received killswitch signal. Quitting.")
		journald.Quit()
	}
	fmt.Println("journald has been shut down...")
}
