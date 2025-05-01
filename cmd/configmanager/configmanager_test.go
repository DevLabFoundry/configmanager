package cmd_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	cfmgrCmd "github.com/dnitsch/configmanager/cmd/configmanager"
)

type cmdRunTestInput struct {
	args        []string
	errored     bool
	exactOutput string
	output      []string
	ctx         context.Context
}

func cmdRunTestHelper(t *testing.T, testInput *cmdRunTestInput) {
	t.Helper()
	ctx := context.TODO()

	if testInput.ctx != nil {
		ctx = testInput.ctx
	}

	cmd := cfmgrCmd.New(ctx)
	os.Args = append([]string{os.Args[0]}, testInput.args...)

	cmd.Cmd.SetArgs(testInput.args)
	errOut := &bytes.Buffer{}
	stdOut := &bytes.Buffer{}
	cmd.Cmd.SetErr(errOut)
	cmd.Cmd.SetOut(stdOut)

	if err := cmd.InitCommand(cfmgrCmd.WithSubCommands()...); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
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
