package cmd_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	cmd "github.com/DevLabFoundry/configmanager/cmd/configmanager"
	"github.com/DevLabFoundry/configmanager/pkg/log"
)

type cmdTestInput struct {
	args        []string
	errored     bool
	exactOutput string
	output      []string
	logLevel    slog.Level // 8 for error -4 debug, 0 for info
}
type levelSetter int

func (l levelSetter) Level() int {
	return 8
}
func cmdRunTestHelper(t *testing.T, testInput *cmdTestInput) {
	t.Helper()

	leveler := &slog.LevelVar{}
	leveler.Set(testInput.logLevel)

	logErr := &bytes.Buffer{}
	logger := log.New(logErr)
	cmd := cmd.NewRootCmd(logger)
	os.Args = append([]string{os.Args[0]}, testInput.args...)
	errOut := &bytes.Buffer{}
	stdOut := &bytes.Buffer{}
	cmd.Cmd.SetArgs(testInput.args)
	cmd.Cmd.SetErr(errOut)
	cmd.Cmd.SetOut(stdOut)

	if err := cmd.Execute(context.TODO()); err != nil {
		if testInput.errored {
			return
		}
		t.Fatalf("\ngot: %v\nwanted <nil>\n", err)
	}

	if testInput.errored && errOut.Len() < 1 {
		t.Errorf("\ngot: nil\nwanted an error to be thrown")
	}
	if len(testInput.output) > 0 {
		for _, v := range testInput.output {
			if !strings.Contains(stdOut.String(), v) {
				t.Errorf("\ngot: %s\vnot found in: %v", stdOut.String(), v)
			}
		}
	}
	if testInput.exactOutput != "" && stdOut.String() != testInput.exactOutput {
		t.Errorf("output mismatch\ngot: %s\n\nwanted: %s", stdOut.String(), testInput.exactOutput)
	}
}
