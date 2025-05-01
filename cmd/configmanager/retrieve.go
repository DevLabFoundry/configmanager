package cmd

import (
	"fmt"

	"github.com/dnitsch/configmanager"
	"github.com/dnitsch/configmanager/internal/cmdutils"
	"github.com/dnitsch/configmanager/pkg/generator"
	"github.com/spf13/cobra"
)

// Default empty string array
var tokenArray []string

type retrieveFlags struct {
	tokens []string
	path   string
}

func newRetrieve(rootCmd *CfgManagerCommand) {
	flags := &retrieveFlags{}
	c := &cobra.Command{
		Use:     "retrieve",
		Aliases: []string{"r", "fetch", "get"},
		Short:   `Retrieves a value for token(s) specified`,
		Long:    `Retrieves a value for token(s) specified and optionally writes to a file or to stdout in a bash compliant export KEY=VAL syntax`,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf := generator.NewConfig().WithTokenSeparator(rootCmd.rootFlags.tokenSeparator).WithOutputPath(flags.path).WithKeySeparator(rootCmd.rootFlags.keySeparator)
			gv := generator.NewGenerator().WithConfig(conf).WithContext(cmd.Context())
			configManager := &configmanager.ConfigManager{}
			return cmdutils.New(gv, configManager).GenerateFromCmd(flags.tokens, flags.path)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(flags.tokens) < 1 {
				return fmt.Errorf("must include at least 1 token")
			}
			return nil
		},
	}
	c.PersistentFlags().StringArrayVarP(&flags.tokens, "token", "t", []string{}, "Token pointing to a config/secret variable. This can be specified multiple times.")
	c.MarkPersistentFlagRequired("token")
	c.PersistentFlags().StringVarP(&flags.path, "path", "p", "./app.env", "Path where to write out the replaced a config/secret variables. Special value of stdout can be used to return the output to stdout e.g. -p stdout, unix style output only")

	rootCmd.Cmd.AddCommand(c)
}
