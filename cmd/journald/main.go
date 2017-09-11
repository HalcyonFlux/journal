package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"os"
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

func init() {
	// TODO: implement flag.Usage
	flag.Usage = func() {}
}

func main() {

	if len(os.Args) == 1 {
		fmt.Println("Please provide a valid journald command")
		flag.Usage()
		os.Exit(1)
	}

	// Subcommands
	srv := flag.NewFlagSet("start-server", flag.ExitOnError)
	clt := flag.NewFlagSet("connect", flag.ExitOnError)

	switch strings.ToLower(os.Args[1]) {

	case "start-server":
		StartServer(srv)

	case "connect":
		StartClient(clt)

	default:
		fmt.Printf("Unknown command '%s'\n", os.Args[1])
		flag.Usage()
		os.Exit(1)
	}

}
