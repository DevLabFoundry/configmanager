package cmd

import (
	"fmt"

	"github.com/dnitsch/configmanager"
	"github.com/dnitsch/configmanager/internal/cmdutils"
	"github.com/dnitsch/configmanager/pkg/generator"
	"github.com/spf13/cobra"
)

type fromStrFlags struct {
	input, path string
}

func newFromString(rootCmd *CfgManagerCommand) {
	flags := &fromStrFlags{}
	c := &cobra.Command{
		Use:     "string-input",
		Aliases: []string{"fromstr", "getfromstr"},
		Short:   `Retrieves all found token values in a specified string input`,
		Long:    `Retrieves all found token values in a specified string input and optionally writes to a file or to stdout in a bash compliant`,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf := generator.NewConfig().WithTokenSeparator(rootCmd.rootFlags.tokenSeparator).WithOutputPath(flags.path).WithKeySeparator(rootCmd.rootFlags.keySeparator)
			gv := generator.NewGenerator().WithConfig(conf).WithContext(cmd.Context())
			configManager := &configmanager.ConfigManager{}
			return cmdutils.New(gv, configManager).GenerateStrOut(flags.input, flags.path)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(flags.input) < 1 {
				return fmt.Errorf("must include input")
			}
			return nil
		},
	}
	c.PersistentFlags().StringVarP(&flags.input, "input", "i", "", `Path to file which contents will be read in or the contents of a string 
inside a variable to be searched for tokens. 
If value is a valid path it will open it if not it will accept the string as an input. 
e.g. -i /some/file or -i $"(cat /som/file)", are both valid input values`)
	c.MarkPersistentFlagRequired("input")
	c.PersistentFlags().StringVarP(&flags.path, "path", "p", "./app.env", `Path where to write out the 
replaced a config/secret variables. Special value of stdout can be used to return the output to stdout e.g. -p stdout, 
unix style output only`)

	rootCmd.Cmd.AddCommand(c)
}
