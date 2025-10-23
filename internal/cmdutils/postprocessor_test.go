package cmdutils_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/generator"
	"github.com/DevLabFoundry/configmanager/v3/internal/cmdutils"
	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
)

func postprocessorHelper(t *testing.T) {
	t.Helper()

}
func Test_ConvertToExportVars(t *testing.T) {
	tests := map[string]struct {
		rawMap       generator.ParsedMap
		expectStr    string
		expectLength int
	}{
		"number included":     {generator.ParsedMap{"foo": "BAR", "num": 123}, `export FOO='BAR'`, 2},
		"strings only":        {generator.ParsedMap{"foo": "BAR", "num": "a123"}, `export FOO='BAR'`, 2},
		"numbers only":        {generator.ParsedMap{"foo": 123, "num": 456}, `export FOO=123`, 2},
		"map inside response": {generator.ParsedMap{"map": `{"foo":"bar","baz":"qux"}`, "num": 123}, `export FOO='bar'`, 3},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			pp := cmdutils.PostProcessor{ProcessedMap: tt.rawMap, Config: config.NewConfig()}
			got := pp.ConvertToExportVar()

			if got == nil {
				t.Errorf(testutils.TestPhrase, got, "not nil")
			}
			if len(got) != tt.expectLength {
				t.Errorf(testutils.TestPhrase, len(got), tt.expectLength)
			}
			st := strings.Join(got, "\n")
			if !strings.Contains(st, tt.expectStr) {
				t.Errorf(testutils.TestPhrase, st, tt.expectStr)
			}

			// check FlushToFile
			tw := bytes.NewBuffer([]byte{})
			pp.FlushOutToFile(tw)
			readBuffer := tw.Bytes()
			if len(readBuffer) == 0 {
				t.Errorf(testutils.TestPhraseWithContext, "buffer should be filled", string(readBuffer), tt.expectStr)
			}

		})
	}
}

func Test_StrToWriter(t *testing.T) {
	ttests := map[string]struct {
		input string
	}{
		"matches":   {`export FOO=BAR`},
		"multiline": {`export FOO=BAR\nBUX=GED`},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			want := tt.input
			tw := bytes.NewBuffer([]byte{})
			pp := cmdutils.PostProcessor{}
			pp.StrToFile(tw, tt.input)
			readBuffer := tw.Bytes()
			if string(readBuffer) != want {
				t.Errorf(testutils.TestPhraseWithContext, "incorrectly written buffer stream", string(readBuffer), want)
			}
		})
	}
}
