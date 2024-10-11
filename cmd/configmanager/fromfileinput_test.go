package cmd_test

import "testing"

func TestFromStr_command(t *testing.T) {

	t.Run("should error on unknown token implementation", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdTestInput{args: []string{"fromstr", "--input", "testdata/input.yml.cm", "--path", "testdata/input.yml"}, errored: false})
	})
	t.Run("should error on missing flag", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdTestInput{args: []string{"fromstr", "--path", "testdata/input.yml"}, errored: true})
	})
}
