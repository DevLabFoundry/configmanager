package cmd

import (
	"github.com/spf13/cobra"
)

type initFlags struct {
}

func newInitCmd(rootCmd *Root) {

	// f := &initFlags{}

	initCmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{},
		Short:   ``,
		Long:    ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			// c := generator.New(cmd.Context(), func(gv *generator.Generator) {
			// 	if rootCmd.rootFlags.verbose {
			// 		rootCmd.logger.SetLevel(log.DebugLvl)
			// 	}
			// 	gv.Logger = rootCmd.logger
			// }).WithConfig(config.NewConfig().WithTokenSeparator(rootCmd.rootFlags.tokenSeparator).WithKeySeparator(rootCmd.rootFlags.keySeparator))

			// // ntm, err := c.DiscoverTokens("")
			// if err != nil {
			// 	return err
			// }

			// initialise pugins here based on discovered tokens
			// //
			// // this can only be done once the tokens are known
			// if err := c.store.Init(cmd.Context(), ntm.TokenSet()); err != nil {
			// 	return nil, fmt.Errorf("%w, %v", ErrProvidersNotFound, err)
			// }
			// defer c.store.PluginCleanUp()
			return nil
		},
	}

	rootCmd.Cmd.AddCommand(initCmd)
}
