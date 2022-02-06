package main

import (
	"context"
	"flag"
	"os"

	"github.com/dnitsch/genvars"
	"github.com/dnitsch/genvars/pkg/log"
)

type tokenArray []string

func (i *tokenArray) String() string {
	return ""
}

func (i *tokenArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	tokens tokenArray
	path   string
)

func main() {
	flag.Parse()
	gv := genvars.NewGenVars(path, context.TODO())
	gv.WithConfig(&genvars.GenVarsConfig{Outpath: path})
	path, err := gv.Generate(tokens)
	if err != nil {
		log.Errorf("%e", err)
		os.Exit(1)
	}
	log.Infof("Vars written to: %s\n", path)
	os.Exit(0)
}

func init() {
	flag.Var(&tokens, "tokens", "token value to look for in specifc implementation")
	flag.StringVar(&path, "path", "./app.env", "Path to write the sourceable file to")
}
