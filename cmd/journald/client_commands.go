package main

import (
	"fmt"
	"github.com/fatih/color"
	uclient "github.com/vaitekunas/unixsock/client"
	"strings"
)

var CMDS = []string{
	"stats - shows journald statistics",
	"create token for <service> <instance> - creates a new journald authentication token",
	"revoke token for <service> <instance> - removes an instance's authentication token",
	"revoke tokens for <service> - removes all service's authentication tokens",
	"list services - lists services using this instance of journald",
	"list instances of <service> - lists all instances of a service using this instance of journald",
	"list remote backends",
	"list logs [number] - lists log files",
	"add remote backend journald <host> <port> <service> <instance> <token> - add a journald backend",
	"remove remote backend journald <host> <port>",
	"",
	"help - prints this information",
	"quit - exits journalist",
}

type client struct {
	unixClient   uclient.UnixSockClient
	unixSockPath string
}

// Run runs a journald client command
func (c *client) Run(cmd string, args map[string]interface{}) {
	resp, err := c.unixClient.Send(cmd, args, true, false)
	if err != nil {
		consoleErr("%s\n", err.Error())
		return
	}
	fmt.Println(resp.Payload)
}

func cmdHelp() {
	blue := color.New(color.FgHiBlue).Sprint
	fmt.Printf("\nAvailable commands:\n\n")
	for _, cmd := range CMDS {
		if cmd == "" {
			fmt.Println("")
			continue
		}
		parts := strings.Split(cmd, "-")
		if len(parts) != 2 {
			continue
		}
		fmt.Printf("\t• %s-%s\n", blue(parts[0]), parts[1])
	}
	fmt.Println("")
}
