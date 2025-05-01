package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type insertFlags struct {
	insertKv map[string]string
}

func newInsertCmd(rootCmd *Root) {
	defaultInsertKv := make(map[string]string)
	f := &insertFlags{}
	insertCmd := &cobra.Command{
		Use:     "insert",
		Aliases: []string{"i", "send", "put"},
		Short:   `Creates the config item in the designated backing store`,
		Long:    ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(f.insertKv) < 1 {
				return fmt.Errorf("must include at least 1 token map")
			}
			return nil
		},
	}
	insertCmd.PersistentFlags().StringToStringVarP(&f.insertKv, "config-pair", "", defaultInsertKv, " token=value pair. This can be specified multiple times.")
	insertCmd.MarkPersistentFlagRequired("config-pair")
	rootCmd.Cmd.AddCommand(insertCmd)
}
