package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/DevLabFoundry/configmanager/v2"
	"github.com/DevLabFoundry/configmanager/v2/internal/cmdutils"
	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/pkg/generator"
	"github.com/DevLabFoundry/configmanager/v2/pkg/log"
	"github.com/spf13/cobra"
)

var (
	Version  string = "0.0.1"
	Revision string = "1111aaaa"
)

type rootCmdFlags struct {
	verbose        bool
	tokenSeparator string
	keySeparator   string
	enableEnvSubst bool
}

type Root struct {
	Cmd       *cobra.Command
	logger    log.ILogger
	rootFlags *rootCmdFlags
}

func NewRootCmd(logger log.ILogger) *Root { //channelOut, channelErr io.Writer
	rc := &Root{
		Cmd: &cobra.Command{
			Use:   config.SELF_NAME,
			Short: fmt.Sprintf("%s CLI for retrieving and inserting config or secret variables", config.SELF_NAME),
			Long: fmt.Sprintf(`%s CLI for retrieving config or secret variables.
			Using a specific tokens as an array item`, config.SELF_NAME),
			SilenceUsage: true,
			Version:      fmt.Sprintf("%s-%s", Version, Revision),
		},
		logger:    logger,
		rootFlags: &rootCmdFlags{},
	}

	rc.Cmd.PersistentFlags().BoolVarP(&rc.rootFlags.verbose, "verbose", "v", false, "Verbosity level")
	rc.Cmd.PersistentFlags().StringVarP(&rc.rootFlags.tokenSeparator, "token-separator", "s", "#", "Separator to use to mark concrete store and the key within it")
	rc.Cmd.PersistentFlags().StringVarP(&rc.rootFlags.keySeparator, "key-separator", "k", "|", "Separator to use to mark a key look up in a map. e.g. AWSSECRETS#/token/map|key1")
	rc.Cmd.PersistentFlags().BoolVarP(&rc.rootFlags.enableEnvSubst, "enable-envsubst", "e", false, "Enable envsubst on input. This will fail on any unset or empty variables")
	addSubCmds(rc)
	return rc
}

// addSubCmds assigns the subcommands to the parent/root command
func addSubCmds(rootCmd *Root) {
	newFromStrCmd(rootCmd)
	newRetrieveCmd(rootCmd)
	newInsertCmd(rootCmd)
}

func (rc *Root) Execute(ctx context.Context) error {
	return rc.Cmd.ExecuteContext(ctx)
}

func cmdutilsInit(rootCmd *Root, cmd *cobra.Command, path string) (*cmdutils.CmdUtils, io.WriteCloser, error) {

	outputWriter, err := cmdutils.GetWriter(path)
	if err != nil {
		return nil, nil, err
	}

	cm := configmanager.New(cmd.Context())
	cm.Config.WithTokenSeparator(rootCmd.rootFlags.tokenSeparator).WithOutputPath(path).WithKeySeparator(rootCmd.rootFlags.keySeparator).WithEnvSubst(rootCmd.rootFlags.enableEnvSubst)
	gnrtr := generator.NewGenerator(cmd.Context(), func(gv *generator.GenVars) {
		if rootCmd.rootFlags.verbose {
			rootCmd.logger.SetLevel(log.DebugLvl)
		}
		gv.Logger = rootCmd.logger
	}).WithConfig(cm.Config)
	cm.WithGenerator(gnrtr)
	return cmdutils.New(cm, rootCmd.logger, outputWriter), outputWriter, nil
}
