package cmd_test

import "testing"

func TestRetrieve_Command(t *testing.T) {
	t.Run("should error on unknown token implementation", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdTestInput{args: []string{"get", "--token", "UNKNOWN://foo/bar", "--token", "UNKNOWN://foo/bar1"}, errored: false})
	})
	t.Run("should error on missing flag", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdTestInput{args: []string{"get"}, errored: true})
	})
}
