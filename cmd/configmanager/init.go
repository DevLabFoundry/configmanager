package cmd

import (
	"strings"

	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/generator"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/spf13/cobra"
)

type initFlags struct {
	tokens []string
}

func newInitCmd(rootCmd *Root) {

	f := &initFlags{}

	initCmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{},
		Short:   `Initialises the plugins required for config retrieval.`,
		Long:    `Initialises the plugins required by creating the relevant folder structure and plugin downloads for the current architecture and operating system`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if f.tokens != nil {
				lt := []string{}
				for _, v := range f.tokens {
					lt = append(lt, strings.ToLower(v))
				}
				c := generator.New(cmd.Context(), func(gv *generator.Generator) {
					if rootCmd.rootFlags.verbose {
						rootCmd.logger.SetLevel(log.DebugLvl)
					}
					gv.Logger = rootCmd.logger
					// still need to parse the root level flags in case the key or token separator is specified
				}).WithConfig(config.NewConfig().WithTokenSeparator(rootCmd.rootFlags.tokenSeparator).WithKeySeparator(rootCmd.rootFlags.keySeparator))

				// NOTE: add additional bells and whistles here to allow for config file parsing and lock management
				return c.InitPlugins(lt)
			}
			// logrus.Debug("zero tokens specified not initialising")
			return nil
		},
	}
	initCmd.PersistentFlags().StringArrayVarP(&f.tokens, "plugin", "", []string{}, `Multi-valued flag to specify the plugins to use with configmanager. 
When not all are specified they will not be pre-initialised and instead will be initialised during the generate run,
NOTE: this may cause issues in more controlled environments
`)
	_ = initCmd.MarkPersistentFlagRequired("plugin")
	rootCmd.Cmd.AddCommand(initCmd)
}
