package main

import (
	"fmt"
	"github.com/fatih/color"
	"strings"
)

var CMDS = []string{
	"stats - shows journald statistics",
	"create token <service> <instance> - creates a new journald authentication token",
	"remove token <service> <instance> - removes an instance's authentication token",
	"remove token <service> - removes all service's authentication tokens",
	"list services - lists services using this instance of journald",
	"list instances of <service> - lists all instances of a service using this instance of journald",
	"list logs - lists log files",
	"",
	"help - prints this information",
	"quit - exits journalist",
}

func cmdHelp() {
	blue := color.New(color.FgHiBlue).Sprint
	fmt.Printf("Available commands:\n\n")
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
}

// cmdGetBanner returns journald's banner
func cmdGetBanner(c *client) {

}

// cmdStats prints journald's statistics
func cmdStats(c *client) {

	cmd := "statistics"
	args := map[string]interface{}{}

	resp, err := c.unixClient.Send(cmd, args, true, false)
	if err != nil {
		fmt.Printf("FAILED: %s\n", err.Error())
		return
	}
	fmt.Println(resp.Payload)

}
