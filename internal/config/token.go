package config

// TokenType is the lexer parsed TokenType
type TokenType string

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	SPACE           TokenType = "SPACE"           // ' '
	TAB             TokenType = "TAB"             // '\t'
	NEW_LINE        TokenType = "NEW_LINE"        // '\n'
	CARRIAGE_RETURN TokenType = "CARRIAGE_RETURN" // '\r'
	CONTROL         TokenType = "CONTROL"

	// Identifiers + literals
	TEXT TokenType = "TEXT"

	EXCLAMATION  TokenType = "!"
	DOUBLE_QUOTE TokenType = "\""
	SINGLE_QUOTE TokenType = "'"
	// other separators
	AT_SIGN             TokenType = "AT_SIGN"             // `@`
	PIPE                TokenType = "PIPE"                // `|`
	COLON               TokenType = "COLON"               // `:`
	EQUALS              TokenType = "EQUALS"              // `=`
	DOT                 TokenType = "DOT"                 // `.`
	COMMA               TokenType = "COMMA"               // `,`
	QUESTION_MARK       TokenType = "QUESTION_MARK"       // `?`
	BACK_SLASH          TokenType = "BACK_SLASH"          // `\`
	FORWARD_SLASH       TokenType = "FORWARD_SLASH"       // `/`
	SLASH_QUESTION_MARK TokenType = "SLASH_QUESTION_MARK" // `/?`

	// Comment Tokens
	DOUBLE_FORWARD_SLASH TokenType = "DOUBLE_FORWARD_SLASH" // `//`
	HASH                 TokenType = "HASH"                 // `#`

	// CONFIGMANAGER_TOKEN Keywords
	// CONFIGMANAGER_TOKEN_SEPARATOR          TokenType = "CONFIGMANAGER_TOKEN_SEPARATOR"          // Dynamically set
	BEGIN_CONFIGMANAGER_TOKEN              TokenType = "BEGIN_CONFIGMANAGER_TOKEN"              // Dynamically set
	CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR TokenType = "CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR" // Dynamically set
	BEGIN_META_CONFIGMANAGER_TOKEN         TokenType = "BEGIN_META_CONFIGMANAGER_TOKEN"         // `[`
	END_META_CONFIGMANAGER_TOKEN           TokenType = "END_META_CONFIGMANAGER_TOKEN"           // `]`
	// This may not possible
	END_CONFIGMANAGER_TOKEN TokenType = "END_CONFIGMANAGER_TOKEN"

	// Parsed "expressions"
	CONFIGMANAGER_TOKEN_CONTENT TokenType = "CONFIGMANAGER_TOKEN_CONTENT"
	UNUSED_TEXT                 TokenType = "UNUSED_TEXT"
)

type Source struct {
	File string `json:"file"`
	Path string `json:"path"`
}

// Token is the basic structure of the captured token
type Token struct {
	Type      TokenType
	Literal   string
	ImpPrefix ImplementationPrefix
	Line      int
	Column    int
	Source    Source
}

var keywords = map[string]TokenType{
	" ":  SPACE,
	"\n": NEW_LINE,
	"\r": CARRIAGE_RETURN,
	"\t": TAB,
	"\f": CONTROL,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TEXT
}

// var typeMapper = map[string]TokenType{
// 	"MESSAGE":   MESSAGE,
// 	"OPERATION": OPERATION,
// 	"CHANNEL":   CHANNEL,
// 	"INFO":      INFO,
// 	"SERVER":    SERVER,
// }

// func LookupType(typ string) TokenType {
// 	if tok, ok := typeMapper[strings.ToUpper(typ)]; ok {
// 		return tok
// 	}
// 	return ""
// }
