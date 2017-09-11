package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

  uclient "github.com/vaitekunas/unixsock/client"
	"github.com/fatih/color"
)

// StartClient starts the journald unix domain socket client
func StartClient(clt *flag.FlagSet) {

	unixSockPathPtr := clt.String("sockfile", "/opt/journald/journald.sock", "path to the journald's unix domain socket file")
	flag.Parse()

	if err := validatePath(*unixSockPathPtr); err != nil {
		fmt.Printf("Invalid path to the unix domain socket file: %s\n", err.Error())
		os.Exit(1)
	}

	c, err := Connect(*unixSockPathPtr)
	if err != nil {
		fmt.Printf("Could not connect to journald: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("Connection to journald established.")
	fmt.Println("Write 'help' for a list of available commands and 'quit' to exit")

	reader := bufio.NewReader(os.Stdin)

	c2 := color.New(color.FgHiBlue)
	prompt := func() {
		fmt.Printf(" %s ", c2.Sprint("◀"))
	}

	// TODO: read ctrl+l
	// TODO: implement > help
Loop:
	for {
		prompt()
		text, _ := reader.ReadString('\n')

		switch strings.ToLower(strings.TrimSpace(text)) {
		case "help":
			cmdHelp()
		case "statistics", "stats":
			cmdStats(c)
		case "clear":
			fmt.Println("\033[H\033[2J")
		case "quit", "exit":
			fmt.Println("exiting")
			break Loop
		default:
			fmt.Printf("\nUnknown command. ")
			cmdHelp()
		}

	}

}


type client struct {
  unixClient uclient.UnixSockClient
  unixSockPath string
}

// Connect instantiates a UnixSockClient
func Connect(unixSockPath string) (*client, error) {
  unixClient, err := uclient.New(unixSockPath)
  if err != nil {
    return nil, fmt.Errorf("Connect: could not instantiate UnixSockClient: %s", err.Error())
  }
  return &client{
    unixClient: unixClient,
    unixSockPath: unixSockPath,
  }, nil
}
