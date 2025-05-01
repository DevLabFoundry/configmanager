package cmd

import (
	"context"
	"fmt"

	"github.com/dnitsch/configmanager/internal/config"
	"github.com/spf13/cobra"
)

var (
	Version  string = "0.0.1"
	Revision string = "1111aaaa"
)

type rootFlags struct {
	verbose        bool
	tokenSeparator string
	keySeparator   string
}

type CfgManagerCommand struct {
	ctx       context.Context
	Cmd       *cobra.Command
	rootFlags *rootFlags
}

func New(ctx context.Context) *CfgManagerCommand {
	flags := &rootFlags{}
	c := &CfgManagerCommand{
		ctx: ctx,
		Cmd: &cobra.Command{
			Use:   config.SELF_NAME,
			Short: fmt.Sprintf("%s CLI for retrieving and inserting config or secret variables", config.SELF_NAME),
			Long: fmt.Sprintf(`%s CLI for retrieving config or secret variables.
			Using a specific tokens as an array item`, config.SELF_NAME),
			Version: fmt.Sprintf("Version: %s\nRevision: %s\n", Version, Revision),
		},
		rootFlags: flags,
	}

	c.Cmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Verbosity level")
	c.Cmd.PersistentFlags().StringVarP(&flags.tokenSeparator, "token-separator", "s", "#", "Separator to use to mark concrete store and the key within it")
	c.Cmd.PersistentFlags().StringVarP(&flags.keySeparator, "key-separator", "k", "|", "Separator to use to mark a key look up in a map. e.g. AWSSECRETS#/token/map|key1")

	return c
}

// WithSubCommands returns a manually maintained list of commands
func WithSubCommands() []func(rootCmd *CfgManagerCommand) {
	// add all sub commands
	return []func(rootCmd *CfgManagerCommand){
		newRetrieve,
		newInsertRun,
		newFromString,
	}
}

// InitCommand ensures each subcommand is added to the root using an IoC injection pattern
func (tc *CfgManagerCommand) InitCommand(iocFuncs ...func(rootCmd *CfgManagerCommand)) error {
	for _, fn := range iocFuncs {
		fn(tc)
	}
	return nil
}

func (c *CfgManagerCommand) Execute() error {
	return c.Cmd.ExecuteContext(c.ctx)
}
