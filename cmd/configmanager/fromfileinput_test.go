package cmd_test

import "testing"

func Test_FromStringCmd(t *testing.T) {
	t.Run("fromstr command called", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:   []string{"fromstr", "-i not_a_token"},
			output: []string{},
		})
	})

	t.Run("fromstr command errors", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"fromstr"},
			output:  []string{},
			errored: true,
		})
	})
}
