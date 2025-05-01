package cmd

import (
	"fmt"

	"github.com/DevLabFoundry/configmanager/v2/internal/cmdutils"
	"github.com/spf13/cobra"
)

type fromStrFlags struct {
	input string
	path  string
}

func newFromStrCmd(rootCmd *Root) {

	f := &fromStrFlags{}

	fromstrCmd := &cobra.Command{
		Use:     "string-input",
		Aliases: []string{"fromstr", "getfromstr"},
		Short:   `Retrieves all found token values in a specified string input`,
		Long:    `Retrieves all found token values in a specified string input, can be a file, stdin, or string. Writes the output to a specified output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputReader, err := cmdutils.GetReader(cmd, f.input)
			if err != nil {
				return err
			}
			cu, outputWriter, err := cmdutilsInit(rootCmd, cmd, f.path)
			if err != nil {
				return err
			}
			defer outputWriter.Close()

			return cu.GenerateStrOut(inputReader, f.input == f.path)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(f.input) < 1 {
				return fmt.Errorf("must include input")
			}
			return nil
		},
	}

	fromstrCmd.PersistentFlags().StringVarP(&f.input, "input", "i", "", `Path to file which contents will be read in or the contents of a string
	inside a variable to be searched for tokens.
	If value is a valid path it will open it if not it will accept the string as an input.
	e.g. -i /some/file or -i $"(cat /som/file)", are both valid input values`)
	fromstrCmd.MarkPersistentFlagRequired("input")
	fromstrCmd.PersistentFlags().StringVarP(&f.path, "path", "p", "./app.env", `Path where to write out the
	replaced a config/secret variables. Special value of stdout can be used to return the output to stdout e.g. -p stdout,
	unix style output only`)
	rootCmd.Cmd.AddCommand(fromstrCmd)
}
