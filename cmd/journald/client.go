package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	uclient "github.com/vaitekunas/unixsock/client"
)

// StartClient starts the journald unix domain socket client
func StartClient(clt *flag.FlagSet) {

	// Subcommand arguments
	unixSockPathPtr := clt.String("sockfile", "/opt/journald/journald.sock", "path to the journald's unix domain socket file")
	clt.Parse(os.Args[2:])

	// Validate UNIX domain socket file
	if err := validatePath(*unixSockPathPtr); err != nil {
		fmt.Printf("Invalid path to the unix domain socket file: %s\n", err.Error())
		os.Exit(1)
	}

	// Connect to the socket
	unixClient, err := uclient.New(*unixSockPathPtr)
	if err != nil {
		consoleErr(fmt.Sprintf("could not instantiate UnixSockClient: %s", err.Error()))
		os.Exit(1)
	}

	c := &client{
		unixClient:   unixClient,
		unixSockPath: *unixSockPathPtr,
	}

	// Say hi
	fmt.Printf("\n%s\n\n", banner)
	message("You are running journald in client mode")
	message("Connection to journald's UNIX domain socket established")
	message("Write 'help' for a list of available commands and 'quit' to exit\n")

	reader := bufio.NewReader(os.Stdin)

	// Command loop
Loop:
	for {
		prompt()
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		lowerText := strings.ToLower(text)
		args := strings.Split(text, " ")

		switch {
		case lowerText == "help":
			cmdHelp()

		case lowerText == "statistics" || lowerText == "stats":
			c.Run("statistics", map[string]interface{}{})

		case argCmd(args, 3) == "create token for":
			c.Run("tokens.add", map[string]interface{}{
				"service":  args[3],
				"instance": args[4],
			})

		case argCmd(args, 3) == "revoke token for":
			c.Run("tokens.revoke.instance", map[string]interface{}{
				"service":  args[3],
				"instance": args[4],
			})

		case argCmd(args, 3) == "revoke tokens for":
			c.Run("tokens.revoke.service", map[string]interface{}{
				"service": args[3],
			})

		case argCmd(args, 3) == "list instances of":
			c.Run("tokens.list.instances", map[string]interface{}{
				"service": args[3],
			})

		case argCmd(args, 2) == "list services":
			c.Run("tokens.list.services", map[string]interface{}{})

		case argCmd(args, 3) == "list remote backends":
			c.Run("remote.list", map[string]interface{}{})

		case argCmd(args, 2) == "list logs":
			if len(args) > 3 {
				c.Run("logs.list", map[string]interface{}{
					"show": args[3],
				})
			} else {
				c.Run("logs.list", map[string]interface{}{})
			}

		case argCmd(args, 4) == "add remote backend journald":
			port, err := strconv.Atoi(args[5])
			if err != nil {
				consoleErr("Invalid port value '%s'", args[5])
			}
			c.Run("remote.add", map[string]interface{}{
				"backend":  "journald",
				"host":     args[4],
				"port":     port,
				"service":  args[6],
				"instance": args[7],
				"token":    args[8],
			})

		case argCmd(args, 4) == "remove remote backend journald":
			port, err := strconv.Atoi(args[5])
			if err != nil {
				consoleErr("Invalid port value '%s'", args[5])
			}
			c.Run("remote.remove", map[string]interface{}{
				"backend": "journald",
				"host":    args[4],
				"port":    port,
			})

		case lowerText == "clear":
			fmt.Println("\033[H\033[2J")

		case lowerText == "quit" || lowerText == "exit":
			fmt.Println("exiting")
			break Loop

		default:
			fmt.Printf("\nUnknown command. ")
			cmdHelp()
		}

	}

}
