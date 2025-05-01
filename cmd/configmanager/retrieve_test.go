package cmd_test

import "testing"

func Test_RetrieveCmd(t *testing.T) {
	t.Run("retrieve command called", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:   []string{"get", "-t not_a_token"},
			output: []string{},
		})
	})
	t.Run("retrieve command errors", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"get"},
			output:  []string{},
			errored: true,
		})
	})
}
