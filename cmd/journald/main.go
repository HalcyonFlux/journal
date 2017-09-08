package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/vaitekunas/journal"
	"github.com/vaitekunas/journal/server"
	"os"
	"os/signal"
	"strings"
)

var banner string

func init() {

	c1 := color.New(color.FgBlue).Sprint
	c2 := color.New(color.FgBlue).Sprint
	c3 := color.New(color.FgBlue).Sprint
	c4 := color.New(color.FgBlue).Sprint
	c5 := color.New(color.FgHiBlue).Sprint
	c6 := color.New(color.FgHiBlue).Sprint
	c7 := color.New(color.FgHiBlue).Sprint
	c8 := color.New(color.FgHiBlue).Sprint
	c9 := color.New(color.FgHiRed).Sprint
	c10 := color.New(color.FgHiRed).Sprint
	c11 := color.New(color.FgHiRed).Sprint
	c12 := color.New(color.FgHiRed).Sprint
	c13 := color.New(color.FgHiWhite).Sprint

	bannerSlice := []string{
		c1(`   d8b                                             888      888`),
		c2(`   Y8P                                             888      888`),
		c3(`                                                   888      888`),
		c4(`  8888  .d88b.  888  888 888d888 88888b.   8888b.  888  .d88888`),
		c5(`  "888 d88""88b 888  888 888P"   888 "88b     "88b 888 d88" 888`),
		c6(`   888 888  888 888  888 888     888  888 .d888888 888 888  888`),
		c7(`   888 Y88..88P Y88b 888 888     888  888 888  888 888 Y88b 888`),
		c8(`   888  "Y88P"   "Y88888 888     888  888 "Y888888 888  "Y88888`),
		c9(`   888`),
		c10(`  d88P`),
		c11(`888P`),
		c12(``),
		c13(VERSION),
	}

	banner = strings.Join(bannerSlice, "\n")
}

func main() {

	// Remote config
	hostPtr := flag.String("host", "127.0.0.1", "Remote logger's host")
	portPtr := flag.Int("port", 4332, "Remote logger's port")
	unixSockPtr := flag.String("unix-socket", "/var/run/logsrv.sock", "Remote logger's unix socket file")
	tokenPtr := flag.String("tokens", "/opt/logsrv/tokens.db", "Remote logger's access tokens")
	statsPtr := flag.String("stats", "/opt/logsrv/stats.db", "Remote logger's statistics")

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

	// Start the local logger
	logSrv, err := server.New(config)
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
		logSrv.Quit()
	case <-logSrv.KillSwitch(): // Can be triggered via the management console
		fmt.Println("Received killswitch signal. Quitting.")
		logSrv.Quit()
	}
	fmt.Println("journald has been shut down...")
}
