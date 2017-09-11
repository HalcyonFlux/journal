package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

func validatePath(path string) error {

	if f, err := os.Stat(path); os.IsNotExist(err) || (err == nil && f.IsDir()) {
		if err != nil {
			return fmt.Errorf("not a valid socket file: %s", err.Error())
		}
		return fmt.Errorf("provided socket file is a directory")
	}

	return nil
}

func prompt() {
	fmt.Printf(" %s ", color.New(color.FgHiBlue).Sprint("◀"))
}

func consoleErr(msg string, a ...interface{}) {
	if len(a) > 0 {
		msg = fmt.Sprintf(msg, a...)
	}
	red := color.New(color.FgHiRed).Sprint
	fmt.Printf(" %s %s\n", red("▲"), red(msg))
}

func message(s string) {
	fmt.Printf(" %s [%s] %s\n", color.New(color.FgHiBlue).Sprint("▶"), time.Now().Format("2006-01-02 15:04:05"), s)
}

// argCmd returns a joined and cleaned command string from args
func argCmd(args []string, length int) string {
	if len(args) < length {
		return ""
	}

	return strings.ToLower(strings.Join(args[:length], " "))
}
