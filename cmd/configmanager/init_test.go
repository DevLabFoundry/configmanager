package cmd_test

import (
	"testing"
)

func Test_Cmd_Init(t *testing.T) {

	t.Run("should initialize AWSSECRETS and AWSPARAMSTR plugins", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdTestInput{args: []string{"init", "--plugin", "AWSPARAMSTR", "--plugin", "AWSSECRETS"}, errored: false})
	})

	t.Run("should error on an empty plugin", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdTestInput{args: []string{"init", "--plugin", "UNKNOWN"}, errored: true})
	})
}
