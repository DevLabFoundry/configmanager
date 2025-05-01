package main

import (
	"context"
	"os"

	cfgmgr "github.com/DevLabFoundry/configmanager/v2/cmd/configmanager"
	"github.com/DevLabFoundry/configmanager/v2/pkg/log"
)

func main() {
	logger := log.New(os.Stderr)
	cmd := cfgmgr.NewRootCmd(logger)
	if err := cmd.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
