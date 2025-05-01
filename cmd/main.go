package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	cfgmgr "github.com/dnitsch/configmanager/cmd/configmanager"
)

func cmdSetUp() (*cfgmgr.CfgManagerCommand, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), []os.Signal{os.Interrupt, syscall.SIGTERM, os.Kill}...)

	cmd := cfgmgr.New(ctx)

	if err := cmd.InitCommand(cfgmgr.WithSubCommands()...); err != nil {
		log.Fatal(err)
	}

	return cmd, stop
}

func main() {
	rootCmd, stop := cmdSetUp()
	defer stop()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
