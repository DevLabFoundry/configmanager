package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/DevLabFoundry/configmanager/v3"
	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/generator"
	"github.com/DevLabFoundry/configmanager/v3/internal/cmdutils"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
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
	// envsubstNoEmpty indicates whether the envsubst no empty flag is enabled
	// only takes effect if enableEnvSubst is true
	envsubstNoEmpty bool
	// laxMode when enabled allows not found keys to be logged out instead of error
	laxMode bool
}

type Root struct {
	Cmd       *cobra.Command
	logger    log.ILogger
	rootFlags *rootCmdFlags
}

func NewRootCmd(logger log.ILogger) *Root { //channelOut, channelErr io.Writer
	var unused bool
	rc := &Root{
		Cmd: &cobra.Command{
			Use:   config.SELF_NAME,
			Short: fmt.Sprintf("%s CLI for retrieving and inserting config or secret variables", config.SELF_NAME),
			Long: fmt.Sprintf(`%s CLI for retrieving config or secret variables.
			Using a specific tokens as an array item`, config.SELF_NAME),
			SilenceUsage:  true,
			SilenceErrors: true,
			Version:       fmt.Sprintf("%s-%s", Version, Revision),
		},
		logger:    logger,
		rootFlags: &rootCmdFlags{},
	}

	rc.Cmd.PersistentFlags().BoolVarP(&rc.rootFlags.verbose, "verbose", "v", false, "Verbosity level")
	rc.Cmd.PersistentFlags().BoolVarP(&unused, "strict", "", true, "")
	rc.Cmd.PersistentFlags().BoolVarP(&rc.rootFlags.laxMode, "no-strict", "", false, "Disable strict mode. By default, strict mode is enabled.")
	rc.Cmd.PersistentFlags().StringVarP(&rc.rootFlags.tokenSeparator, "token-separator", "s", "://", "Separator to use to mark concrete store and the key within it")
	rc.Cmd.PersistentFlags().StringVarP(&rc.rootFlags.keySeparator, "key-separator", "k", "|", "Separator to use to mark a key look up in a map. e.g. AWSSECRETS:///token/map|key1")
	rc.Cmd.PersistentFlags().BoolVarP(&rc.rootFlags.enableEnvSubst, "enable-envsubst", "e", false, "Enable envsubst on input. This will fail on any unset variables")
	rc.Cmd.PersistentFlags().BoolVarP(&rc.rootFlags.enableEnvSubst, "envsubst-no-empty", "", false, "Enable envsubst no empty check. This will fail on any unset and/or empty variables")

	// Mark the --strict flag as deprecated since it will be removed in future versions
	_ = rc.Cmd.PersistentFlags().MarkDeprecated("strict", "The behaviour is strict by default, you can disable this by specifying --no-strict")
	// rc.Cmd.Flags().SetOutput(os.Stderr)
	addSubCmds(rc)
	return rc
}

// addSubCmds assigns the subcommands to the parent/root command
func addSubCmds(rootCmd *Root) {
	newFromStrCmd(rootCmd)
	newRetrieveCmd(rootCmd)
	newInsertCmd(rootCmd)
	newInitCmd(rootCmd)
}

func (rc *Root) Execute(ctx context.Context) error {
	return rc.Cmd.ExecuteContext(ctx)
}

func cmdutilsInit(rootCmd *Root, cmd *cobra.Command, path string) (*cmdutils.CmdUtils, io.WriteCloser, error) {

	outputWriter, err := cmdutils.GetWriter(path)
	if err != nil {
		return nil, nil, err
	}

	if rootCmd.rootFlags.verbose {
		rootCmd.logger.SetLevel(log.DebugLvl)
	}
	cm := configmanager.New(cmd.Context())
	cm.WithLogger(rootCmd.logger)
	cm.Config.WithTokenSeparator(rootCmd.rootFlags.tokenSeparator).
		WithOutputPath(path).
		WithKeySeparator(rootCmd.rootFlags.keySeparator).
		WithEnvSubst(rootCmd.rootFlags.enableEnvSubst).
		WithLaxMode(rootCmd.rootFlags.laxMode).
		WithEnvSubstNoEmpty(rootCmd.rootFlags.envsubstNoEmpty)
	gnrtr := generator.New(cmd.Context(), func(gv *generator.Generator) {
		gv.Logger = rootCmd.logger
	}).
		WithConfig(cm.Config).
		WithStores(store.New(cmd.Context(), store.WithLogger(rootCmd.logger)))

	cm.WithGenerator(gnrtr)
	return cmdutils.New(cm, rootCmd.logger, outputWriter), outputWriter, nil
}
