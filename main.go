package main

import (
	"github.com/pon-pbs/bbRelay/cmd"
	_ "github.com/pon-pbs/bbRelay/docs"
)

var RelayVersion = "dev"

func main() {
	cmd.RelayVersion = RelayVersion
	cmd.Execute()
}