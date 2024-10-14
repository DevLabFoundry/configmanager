package cmdutils_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dnitsch/configmanager/internal/cmdutils"
	"github.com/dnitsch/configmanager/internal/config"
	"github.com/dnitsch/configmanager/internal/testutils"
	"github.com/dnitsch/configmanager/pkg/generator"
	log "github.com/dnitsch/configmanager/pkg/log"
	"github.com/spf13/cobra"
)

type mockCfgMgr struct {
	parsedMap    generator.ParsedMap
	err          error
	parsedString string
	config       *config.GenVarsConfig
}

func (m mockCfgMgr) RetrieveWithInputReplaced(input string) (string, error) {
	return m.parsedString, m.err
}

func (m mockCfgMgr) Retrieve(tokens []string) (generator.ParsedMap, error) {
	return m.parsedMap, m.err
}

func (m mockCfgMgr) GeneratorConfig() *config.GenVarsConfig {
	return m.config
}

func Test_UploadTokens_errors(t *testing.T) {
	m := &mockCfgMgr{}
	cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), io.Discard)
	tokenMap := make(map[string]string)
	if err := cmd.UploadTokensWithVals(tokenMap); err == nil {
		t.Errorf(testutils.TestPhraseWithContext, "NOT YET IMPLEMENTED should fail", nil, "err")
	}
}

func cmdTestHelper(t *testing.T, err error, got []byte, expect []string) {
	t.Helper()
	if err != nil {
		t.Errorf("wanted file to not Error")
	}

	if len(got) < 1 {
		t.Error("empty file")
	}
	for _, want := range expect {
		if !strings.Contains(string(got), want) {
			t.Errorf(testutils.TestPhraseWithContext, "contents not found", string(got), want)
		}
	}
}

func Test_GenerateFromCmd(t *testing.T) {
	ttests := map[string]struct {
		mockMap generator.ParsedMap
		tokens  []string
		expect  []string
	}{
		"succeeds with 3 tokens": {
			generator.ParsedMap{"FOO://bar/qusx": "aksujg", "FOO://bar/lorem": "", "FOO://bar/ducks": "sdhbjk0293"},
			[]string{"FOO://bar/qusx", "FOO://bar/lorem", "FOO://bar/ducks"},
			[]string{"export QUSX='aksujg'", "export LOREM=''", "export DUCKS='sdhbjk0293'"},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			// create a temp file
			f, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-token*")
			defer os.Remove(f.Name())

			m := &mockCfgMgr{
				config:    config.NewConfig(),
				parsedMap: tt.mockMap,
			}

			cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), f)
			err := cmd.GenerateFromCmd(tt.tokens)
			if err != nil {
				t.Fatalf(testutils.TestPhraseWithContext, "generate from cmd tokens", err, nil)
			}

			got, err := io.ReadAll(f)
			cmdTestHelper(t, err, got, tt.expect)
		})
	}
}

type mockWriter struct {
	w io.Writer
}

func (m *mockWriter) Write(in []byte) (int, error) {
	return m.w.Write(in)
}

func Test_GenerateStrOut(t *testing.T) {

	inputStr := `FOO://bar/qusx FOO://bar/lorem FOO://bar/ducks`
	mockParsedStr := `aksujg fooLorem Mighty`
	expect := []string{"aksujg", "fooLorem", "Mighty"}

	t.Run("succeeds with input from string and output different", func(t *testing.T) {
		tearDown, reader, file := func(t *testing.T) (func(), io.Reader, string) {
			f, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-string*")
			return func() {
				os.Remove(f.Name())
			}, f, f.Name()
		}(t)
		defer tearDown()

		m := &mockCfgMgr{
			config:       config.NewConfig(),
			parsedString: mockParsedStr,
		}
		inputReader, _ := cmdutils.GetReader(&cobra.Command{}, inputStr)
		outputWriter, _ := cmdutils.GetWriter(file)

		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), outputWriter)
		err := cmd.GenerateStrOut(inputReader, false)
		if err != nil {
			t.Fatalf(testutils.TestPhraseWithContext, "generate from string", err, nil)
		}
		got, err := io.ReadAll(reader)
		cmdTestHelper(t, err, got, expect)
	})

	t.Run("succeeds output set to stdout", func(t *testing.T) {
		m := &mockCfgMgr{
			config:       config.NewConfig(),
			parsedString: mockParsedStr,
		}
		inputReader, _ := cmdutils.GetReader(&cobra.Command{}, inputStr)
		outputWriter, _ := cmdutils.GetWriter("stdout")
		mw := &mockWriter{w: outputWriter}

		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), mw)
		writer := bytes.NewBuffer([]byte{})

		err := cmd.GenerateStrOut(inputReader, false)
		if err != nil {
			t.Fatalf(testutils.TestPhraseWithContext, "generate from string", err, nil)
		}
		got, err := io.ReadAll(writer)
		cmdTestHelper(t, err, got, expect)
	})
	t.Run("succeeds input and output are set to file names", func(t *testing.T) {
		inputF, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-string*")
		inputF.Write([]byte(inputStr))
		outputF, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-string*")

		defer func() {
			os.Remove(inputF.Name())
			os.Remove(outputF.Name())
		}()

		// inputReader, _ := cmdutils.GetReader(&cobra.Command{}, inputStr)
		// outputWriter, _ := cmdutils.GetWriter("stdout")

		m := &mockCfgMgr{
			config:       config.NewConfig(),
			parsedString: mockParsedStr,
		}

		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), outputF)
		err := cmd.GenerateStrOut(inputF, false)
		if err != nil {
			t.Fatalf(testutils.TestPhraseWithContext, "generate from string", err, nil)
		}
		got, err := io.ReadAll(outputF)
		cmdTestHelper(t, err, got, expect)
	})

	t.Run("succeeds input and output are set to the same file", func(t *testing.T) {
		inputF, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-string*")
		inputF.Write([]byte(inputStr))
		// outputF, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-string*")
		defer func() {
			os.Remove(inputF.Name())
		}()

		m := &mockCfgMgr{
			config:       config.NewConfig().WithOutputPath(inputF.Name()),
			parsedString: mockParsedStr,
		}
		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), inputF)
		err := cmd.GenerateStrOut(inputF, true)
		if err != nil {
			t.Fatalf(testutils.TestPhraseWithContext, "generate from string", err, nil)
		}
		got, err := os.ReadFile(inputF.Name())
		cmdTestHelper(t, err, got, expect)
	})
}

func Test_CmdUtils_Errors_on(t *testing.T) {
	t.Run("outputFile wrong", func(t *testing.T) {
		// m := &mockCfgMgr{
		// 	config:    config.NewConfig(),
		// 	parsedMap: generator.ParsedMap{"FOO://bar/qusx": "aksujg", "FOO://bar/lorem": "", "FOO://bar/ducks": "sdhbjk0293"},
		// }

		// inputReader, _ := cmdutils.GetReader(&cobra.Command{}, inputStr)
		_, err := cmdutils.GetWriter("xunknown/file")
		if err == nil {
			t.Fatal("error not caught")
		}
		// cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), outputWriter)
		// if err := cmd.GenerateFromCmd([]string{"IMNP://foo"}, "xunknown/file"); err == nil {
		// 	t.Errorf(testutils.TestPhraseWithContext, "file does not exist unable to create a writer", "err", nil)
		// }
	})
	t.Run("REtrieve from tokens in fetching ANY of the tokens", func(t *testing.T) {
		m := &mockCfgMgr{
			config:    config.NewConfig(),
			parsedMap: generator.ParsedMap{},
			err:       fmt.Errorf("err in fetching tokens"),
		}

		writer := bytes.NewBuffer([]byte{})
		mw := &mockWriter{w: writer}
		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), mw)
		if err := cmd.GenerateFromCmd([]string{"IMNP://foo"}); err == nil {
			t.Errorf(testutils.TestPhraseWithContext, "NOT fetching ANY tokens should error", "err", nil)
		}
	})

	t.Run("REtrieve from tokens in fetching SOME of the tokens", func(t *testing.T) {
		m := &mockCfgMgr{
			config:    config.NewConfig(),
			parsedMap: generator.ParsedMap{"IMNP://foo": "bar"},
			err:       fmt.Errorf("err in fetching tokens"),
		}

		writer := bytes.NewBuffer([]byte{})
		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), &mockWriter{w: writer})
		if err := cmd.GenerateFromCmd([]string{"IMNP://foo", "IMNP://foo2"}); err != nil {
			t.Errorf(testutils.TestPhraseWithContext, "fetching tokens some erroring should only be logged out", "err", nil)
		}
	})

	t.Run("REtrieve from string in fetching SOME of the tokens", func(t *testing.T) {
		m := &mockCfgMgr{
			config:       config.NewConfig().WithOutputPath("stdout"),
			parsedMap:    generator.ParsedMap{"IMNP://foo": "bar"},
			parsedString: `bar `,
			err:          fmt.Errorf("err in fetching tokens"),
		}

		inputReader, _ := cmdutils.GetReader(&cobra.Command{}, `"IMNP://foo", "IMNP://foo2"`)

		writer := bytes.NewBuffer([]byte{})
		mw := &mockWriter{w: writer}
		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), mw)
		if err := cmd.GenerateStrOut(inputReader, false); err == nil {
			t.Errorf(testutils.TestPhraseWithContext, "fetching tokens some erroring should only be logged out", nil, "err")
		}
	})

	t.Run("REtrieve from string in fetching SOME of the tokens with input/output the same", func(t *testing.T) {
		inputF, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-string*")
		inputF.Write([]byte(`"IMNP://foo", "IMNP://foo2"`))
		// outputF, _ := os.CreateTemp(os.TempDir(), "gen-conf-frrom-string*")
		defer func() {
			os.Remove(inputF.Name())
		}()

		m := &mockCfgMgr{
			config:       config.NewConfig().WithOutputPath(inputF.Name()),
			parsedString: `bar `,
			err:          fmt.Errorf("err in fetching tokens"),
		}

		writer := bytes.NewBuffer([]byte{})
		mw := &mockWriter{w: writer}
		cmd := cmdutils.New(m, log.New(&bytes.Buffer{}), mw)
		if err := cmd.GenerateStrOut(inputF, true); err == nil {
			t.Errorf(testutils.TestPhraseWithContext, "fetching tokens some erroring should only be logged out", nil, "err")
		}
	})
}
