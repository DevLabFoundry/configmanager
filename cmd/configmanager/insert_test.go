package cmd_test

import "testing"

func Test_InsertCmd(t *testing.T) {
	t.Run("insert command called", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"insert", "-t token=val"},
			output:  []string{"not yet implemented"},
			errored: true,
		})
	})
}
