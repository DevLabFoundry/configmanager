package main

import (
	"context"
	"fmt"
	"os"

	cfgmgr "github.com/DevLabFoundry/configmanager/v2/cmd/configmanager"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
)

const redTerminal = "\x1b[31m%s\x1b[0m"

func main() {
	logger := log.New(os.Stderr)
	cmd := cfgmgr.NewRootCmd(logger)
	if err := cmd.Execute(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, redTerminal+"\n", err)
		os.Exit(1)
	}
}
