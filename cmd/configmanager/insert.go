package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type insertFlags struct {
	insertKv map[string]string
}

func newInsertRun(rootCmd *CfgManagerCommand) {
	defaultInsertKv := map[string]string{}
	flags := &insertFlags{}
	c := &cobra.Command{
		Use:     "insert",
		Aliases: []string{"i", "send", "put"},
		Short:   `Retrieves a value for token(s) specified and optionally writes to a file`,
		Long:    `Retrieves a value for token(s) specified and optionally writes to a file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}
	c.PersistentFlags().StringToStringVarP(&flags.insertKv, "item", "t", defaultInsertKv, "Token pointing to a config/secret variable. This can be specified multiple times.")
	c.MarkPersistentFlagRequired("item")
	rootCmd.Cmd.AddCommand(c)
}
