package lexer_test

import (
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/lexer"
)

func Test_Lexer_NextToken(t *testing.T) {
	input := `foo stuyfsdfsf
foo=AWSPARAMSTR:///path|keyAWSSECRETS:///foo
META_INCLUDED=VAULT://baz/bar/123|key1.prop2[role=arn:aws:iam::1111111:role,version=1082313]
`
	ttests := []struct {
		expectedType    config.TokenType
		expectedLiteral string
	}{
		{config.TEXT, "foo"},
		{config.SPACE, " "},
		{config.TEXT, "stuyfsdfsf"},
		{config.NEW_LINE, "\n"},
		{config.TEXT, "foo"},
		{config.EQUALS, "="},
		{config.BEGIN_CONFIGMANAGER_TOKEN, "AWSPARAMSTR://"},
		{config.FORWARD_SLASH, "/"},
		{config.TEXT, "path"},
		{config.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR, "|"},
		{config.TEXT, "key"},
		{config.BEGIN_CONFIGMANAGER_TOKEN, "AWSSECRETS://"},
		{config.FORWARD_SLASH, "/"},
		{config.TEXT, "foo"},
		{config.NEW_LINE, "\n"},
		{config.TEXT, "MET"},
		{config.TEXT, "A"},
		{config.TEXT, "_INCL"},
		{config.TEXT, "U"},
		{config.TEXT, "DED"},
		{config.EQUALS, "="},
		{config.BEGIN_CONFIGMANAGER_TOKEN, "VAULT://"},
		{config.TEXT, "baz"},
		{config.FORWARD_SLASH, "/"},
		{config.TEXT, "bar"},
		{config.FORWARD_SLASH, "/"},
		{config.TEXT, "123"},
		{config.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR, "|"},
		{config.TEXT, "key1"},
		{config.DOT, "."},
		{config.TEXT, "prop2"},
		{config.BEGIN_META_CONFIGMANAGER_TOKEN, "["},
		{config.TEXT, "role"},
		{config.EQUALS, "="},
		{config.TEXT, "arn"},
		{config.COLON, ":"},
		{config.TEXT, "aws"},
		{config.COLON, ":"},
		{config.TEXT, "iam"},
		{config.COLON, ":"},
		{config.COLON, ":"},
		{config.TEXT, "1111111"},
		{config.COLON, ":"},
		{config.TEXT, "role"},
		{config.COMMA, ","},
		{config.TEXT, "version"},
		{config.EQUALS, "="},
		{config.TEXT, "1082313"},
		{config.END_META_CONFIGMANAGER_TOKEN, "]"},
		{config.NEW_LINE, "\n"},
		{config.EOF, ""},
	}

	l := lexer.New(lexer.Source{Input: input, FullPath: "/foo/bar", FileName: "bar"}, *config.NewConfig())

	for i, tt := range ttests {

		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. got=%q, expected=%q",
				i, tok.Type, tt.expectedType)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. got=%q, expected=%q",
				i, tok.Literal, tt.expectedLiteral)
		}
		if tok.Type == config.BEGIN_CONFIGMANAGER_TOKEN {

		}
	}
}

func Test_empty_file(t *testing.T) {
	input := ``
	l := lexer.New(lexer.Source{Input: input, FullPath: "/foo/bar", FileName: "bar"}, *config.NewConfig())
	tok := l.NextToken()
	if tok.Type != config.EOF {
		t.Fatal("expected EOF")
	}
}
