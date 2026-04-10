package lexer_test

import (
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/lexer"
	"github.com/DevLabFoundry/configmanager/v3/internal/token"
)

func Test_Lexer_NextToken(t *testing.T) {
	input := `foo stuyfsdfsf
foo=AWSPARAMSTR:///path|keyAWSSECRETS:///foo
META_INCLUDED=VAULT://baz/bar/123|key1.prop2[role=arn:aws:iam::1111111:role,version=1082313]
`
	ttests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.TEXT, "foo"},
		{token.SPACE, " "},
		{token.TEXT, "stuyfsdfsf"},
		{token.NEW_LINE, "\n"},
		{token.TEXT, "foo"},
		{token.EQUALS, "="},
		{token.BEGIN_CONFIGMANAGER_TOKEN, "AWSPARAMSTR://"},
		{token.FORWARD_SLASH, "/"},
		{token.TEXT, "path"},
		{token.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR, "|"},
		{token.TEXT, "key"},
		{token.BEGIN_CONFIGMANAGER_TOKEN, "AWSSECRETS://"},
		{token.FORWARD_SLASH, "/"},
		{token.TEXT, "foo"},
		{token.NEW_LINE, "\n"},
		{token.TEXT, "MET"},
		{token.TEXT, "A"},
		{token.TEXT, "_INCLUDED"},
		// {token.TEXT, "U"},
		// {token.TEXT, "DED"},
		{token.EQUALS, "="},
		{token.BEGIN_CONFIGMANAGER_TOKEN, "VAULT://"},
		{token.TEXT, "baz"},
		{token.FORWARD_SLASH, "/"},
		{token.TEXT, "bar"},
		{token.FORWARD_SLASH, "/"},
		{token.TEXT, "123"},
		{token.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR, "|"},
		{token.TEXT, "key1"},
		{token.DOT, "."},
		{token.TEXT, "prop2"},
		{token.BEGIN_META_CONFIGMANAGER_TOKEN, "["},
		{token.TEXT, "role"},
		{token.EQUALS, "="},
		{token.TEXT, "arn"},
		{token.COLON, ":"},
		{token.TEXT, "aws"},
		{token.COLON, ":"},
		{token.TEXT, "iam"},
		{token.COLON, ":"},
		{token.COLON, ":"},
		{token.TEXT, "1111111"},
		{token.COLON, ":"},
		{token.TEXT, "role"},
		{token.COMMA, ","},
		{token.TEXT, "version"},
		{token.EQUALS, "="},
		{token.TEXT, "1082313"},
		{token.END_META_CONFIGMANAGER_TOKEN, "]"},
		{token.NEW_LINE, "\n"},
		{token.EOF, ""},
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
		if tok.Type == token.BEGIN_CONFIGMANAGER_TOKEN {

		}
	}
}

func Test_empty_file(t *testing.T) {
	input := ``
	l := lexer.New(lexer.Source{Input: input, FullPath: "/foo/bar", FileName: "bar"}, *config.NewConfig())
	tok := l.NextToken()
	if tok.Type != token.EOF {
		t.Fatal("expected EOF")
	}
}
