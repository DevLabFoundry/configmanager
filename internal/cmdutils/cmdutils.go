// pacakge Cmdutils
//
// Wraps around the ConfigManager library
// with additional postprocessing capabilities for
// output management when used with cli flags.
package cmdutils

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/generator"
	"github.com/spf13/cobra"
)

type configManagerIface interface {
	RetrieveWithInputReplaced(input string) (string, error)
	Retrieve(tokens []string) (generator.ParsedMap, error)
	GeneratorConfig() *config.GenVarsConfig
}

type CmdUtils struct {
	logger           log.ILogger
	configManager    configManagerIface
	outputWriter     io.WriteCloser
	tempOutputWriter io.WriteCloser
}

func New(confManager configManagerIface, logger log.ILogger, writer io.WriteCloser) *CmdUtils {
	return &CmdUtils{
		logger:        logger,
		configManager: confManager,
		outputWriter:  writer,
	}
}

// GenerateFromTokens is a helper cmd method to call from retrieve command
func (c *CmdUtils) GenerateFromCmd(tokens []string) error {
	return c.generateFromToken(tokens)
}

// generateFromToken
func (c *CmdUtils) generateFromToken(tokens []string) error {
	pm, err := c.configManager.Retrieve(tokens)
	if err != nil {
		// return full error to terminal if no tokens were parsed
		if len(pm) < 1 {
			return err
		}
		c.logger.Error("%v", err)
	}
	// Conver to ExportVars and flush to file
	pp := &PostProcessor{ProcessedMap: pm, Config: c.configManager.GeneratorConfig()}
	pp.ConvertToExportVar()
	return pp.FlushOutToFile(c.outputWriter)
}

// Generate a replaced string from string input command
//
// returns a non empty string if move of temp file is required
func (c *CmdUtils) GenerateStrOut(input io.Reader, inputOutputIsSame bool) error {

	// outputs and inputs match and are file paths
	if inputOutputIsSame {
		c.logger.Debug("overwrite mode on")
		// create a temp file
		tempfile, err := os.CreateTemp(os.TempDir(), "configmanager")
		if err != nil {
			return err
		}
		defer os.Remove(tempfile.Name())
		c.logger.Debug("tmp file created: %s", tempfile.Name())
		c.tempOutputWriter = tempfile
		defer c.tempOutputWriter.Close()
		return c.generateFromStrOutOverwrite(input, tempfile.Name())
	}

	return c.generateStrOutFromInput(input, c.outputWriter)
}

// generateFromStrOutOverwrite uses the same file for input as output
// requires additional consideration and must create a temp file
// and then write contents from temp to actual target
// otherwise, two open file operations would be targeting same descriptor
// will cause issues and inconsistent writes
func (c *CmdUtils) generateFromStrOutOverwrite(input io.Reader, outtemp string) error {

	if err := c.generateStrOutFromInput(input, c.tempOutputWriter); err != nil {
		return err
	}
	tr, err := os.ReadFile(outtemp)
	if err != nil {
		return err
	}
	// move temp file to output path
	if _, err := c.outputWriter.Write(tr); err != nil {
		return err
	}
	return nil
}

// generateStrOutFromInput takes a reader and writer as input
func (c *CmdUtils) generateStrOutFromInput(input io.Reader, writer io.Writer) error {

	b, err := io.ReadAll(input)
	if err != nil {
		return err
	}

	str, err := c.configManager.RetrieveWithInputReplaced(string(b))
	if err != nil {
		return err
	}
	pp := &PostProcessor{}

	return pp.StrToFile(writer, str)
}

type WriterCloserWrapper struct {
	io.Writer
}

func (swc *WriterCloserWrapper) Close() error {
	return nil
}

func GetWriter(outputpath string) (io.WriteCloser, error) {
	outputWriter := &WriterCloserWrapper{os.Stdout}
	if outputpath != "stdout" {
		return os.Create(outputpath)
	}
	return outputWriter, nil
}

func GetReader(cmd *cobra.Command, inputpath string) (io.Reader, error) {
	inputReader := cmd.InOrStdin()
	if inputpath != "-" && inputpath != "" {
		if _, err := os.Stat(inputpath); os.IsNotExist(err) {
			return strings.NewReader(inputpath), nil
		}
		return os.Open(inputpath)
	}
	return inputReader, nil
}

// UploadTokensWithVals takes in a map of key/value pairs and uploads them
func (c *CmdUtils) UploadTokensWithVals(tokens map[string]string) error {
	return fmt.Errorf("notYetImplemented")
}
