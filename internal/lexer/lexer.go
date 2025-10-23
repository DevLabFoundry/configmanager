// Package lexer
//
// Performs lexical analysis on the source files and emits tokens.
package lexer

import (
	"github.com/DevLabFoundry/configmanager/v3/internal/config"
)

// nonText characters captures all character sets that are _not_ assignable to TEXT
var nonText = map[string]bool{
	// separators
	" ": true, "\n": true, "\r": true, "\t": true,
	"=": true, ".": true, ",": true, "|": true, "?": true, "/": true, "@": true, ":": true,
	"]": true, "[": true, "'": true, "\"": true,
	// initial chars of potential identifiers
	// this forces the lexer to not treat at as TEXT
	// and enter the switch statement of the state machine
	// NOTE: when a new implementation is added we should add it here
	// AWS|AZure...
	"A": true,
	// VAULT (HashiCorp)
	"V": true,
	// GCP
	"G": true,
	// Unknown
	"U": true,
}

type Source struct {
	Input    string
	FileName string
	FullPath string
}

// Lexer
type Lexer struct {
	config       config.GenVarsConfig
	keySeparator byte
	length       int
	source       Source
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line - start at 1
	column       int  // column of text - gets set to 0 on every new line - start at 0
}

// New returns a Lexer pointer allocation
func New(source Source, config config.GenVarsConfig) *Lexer {
	l := &Lexer{
		source:       source,
		line:         1,
		column:       0,
		length:       len(source.Input),
		config:       config,
		keySeparator: config.KeySeparator()[0],
	}
	l.readChar()
	return l
}

// NextToken advances through the source returning a found token
func (l *Lexer) NextToken() config.Token {
	var tok config.Token

	switch l.ch {
	// identify the dynamically selected key separator
	case l.keySeparator:
		tok = config.Token{Type: config.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR, Literal: string(l.ch)}
	// Specific cases for BEGIN_CONFIGMANAGER_TOKEN possibilities
	case 'A':
		if l.peekChar() == 'W' {
			// AWS store types
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.SecretMgrPrefix, config.ParamStorePrefix}, "AW"); found {
				tok = config.Token{Type: config.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker AW as text
				tok = config.Token{Type: config.TEXT, Literal: "AW"}
			}
		} else if l.peekChar() == 'Z' {
			// Azure Store Types
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.AzKeyVaultSecretsPrefix, config.AzTableStorePrefix, config.AzAppConfigPrefix}, "AZ"); found {
				tok = config.Token{Type: config.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker AZ as text
				tok = config.Token{Type: config.TEXT, Literal: "AZ"}
			}
		} else {
			tok = config.Token{Type: config.TEXT, Literal: "A"}
		}
	case 'G':
		// GCP TOKENS
		if l.peekChar() == 'C' {
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.GcpSecretsPrefix}, "GC"); found {
				tok = config.Token{Type: config.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker AW as text
				tok = config.Token{Type: config.TEXT, Literal: "GC"}
			}
		} else {
			tok = config.Token{Type: config.TEXT, Literal: "G"}
		}
	case 'V':
		// HASHI VAULT Tokens
		if l.peekChar() == 'A' {
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.HashicorpVaultPrefix}, "VA"); found {
				tok = config.Token{Type: config.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker VA as text
				tok = config.Token{Type: config.TEXT, Literal: "VA"}
			}
		} else {
			tok = config.Token{Type: config.TEXT, Literal: "V"}
		}
	case 'U':
		// UNKNOWN
		if l.peekChar() == 'N' {
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.UnknownPrefix}, "UN"); found {
				tok = config.Token{Type: config.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker AW as text
				tok = config.Token{Type: config.TEXT, Literal: "UN"}
			}
		} else {
			tok = config.Token{Type: config.TEXT, Literal: "U"}
		}
	case '=':
		tok = config.Token{Type: config.EQUALS, Literal: "="}
	case '.':
		tok = config.Token{Type: config.DOT, Literal: "."}
	case ',':
		tok = config.Token{Type: config.COMMA, Literal: ","}
	case '/':
		if l.peekChar() == '?' {
			l.readChar()
			tok = config.Token{Type: config.SLASH_QUESTION_MARK, Literal: "/?"}
		} else {
			tok = config.Token{Type: config.FORWARD_SLASH, Literal: "/"}
		}
	case '\\':
		tok = config.Token{Type: config.BACK_SLASH, Literal: "\\"}
	case '?':
		tok = config.Token{Type: config.QUESTION_MARK, Literal: "?"}
	case ']':
		tok = config.Token{Type: config.END_META_CONFIGMANAGER_TOKEN, Literal: "]"}
	case '[':
		tok = config.Token{Type: config.BEGIN_META_CONFIGMANAGER_TOKEN, Literal: "["}
	case '|':
		tok = config.Token{Type: config.PIPE, Literal: "|"}
	case '@':
		tok = config.Token{Type: config.AT_SIGN, Literal: "@"}
	case ':':
		tok = config.Token{Type: config.COLON, Literal: ":"}
	case '"':
		tok = config.Token{Type: config.DOUBLE_QUOTE, Literal: "\""}
	case '\'':
		tok = config.Token{Type: config.SINGLE_QUOTE, Literal: "'"}
	case '\n':
		l.line = l.line + 1
		l.column = 0 // reset column count
		tok = l.setTextSeparatorToken()
		// want to preserve all indentations and punctuation
	case ' ', '\r', '\t', '\f':
		tok = l.setTextSeparatorToken()
	case 0:
		tok.Literal = ""
		tok.Type = config.EOF
	// case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	// 	'B', 'C', 'D', 'E', 'F', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'W', 'X', 'Y', 'Z',
	// 	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
	default:
		if isText(l.ch) {
			tok.Literal = l.readText()
			tok.Type = config.TEXT
			return tok
		}
		tok = newToken(config.ILLEGAL, l.ch)
	}
	// add general properties to each token
	tok.Line = l.line
	tok.Column = l.column
	tok.Source = config.Source{Path: l.source.FullPath, File: l.source.FileName}
	l.readChar()
	return tok
}

// readChar moves cursor along
func (l *Lexer) readChar() {
	if l.readPosition >= l.length {
		l.ch = 0
	} else {
		l.ch = l.source.Input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
	l.column += 1
}

// peekChar reveals next char withouh advancing the cursor along
func (l *Lexer) peekChar() byte {
	if l.readPosition >= l.length {
		return 0
	} else {
		return l.source.Input[l.readPosition]
	}
}

func (l *Lexer) readText() string {
	position := l.position
	for isText(l.ch) && l.readPosition <= l.length {
		l.readChar()
	}
	return l.source.Input[position:l.position]
}

func (l *Lexer) setTextSeparatorToken() config.Token {
	tok := newToken(config.LookupIdent(string(l.ch)), l.ch)
	return tok
}

// peekIsBeginOfToken attempts to identify the gendoc keyword after 2 slashes
func (l *Lexer) peekIsBeginOfToken(possibleBeginToken []config.ImplementationPrefix, charsRead string) (bool, string, config.ImplementationPrefix) {
	for _, pbt := range possibleBeginToken {
		configToken := ""
		pbtWithTokenSep := string(pbt[len(charsRead):]) + l.config.TokenSeparator()
		for i := 0; i < len(pbtWithTokenSep); i++ {
			configToken += string(l.peekChar())
			l.readChar()
		}

		if configToken == pbtWithTokenSep {
			return true, charsRead + configToken, pbt
		}
		l.resetAfterPeek(len(pbtWithTokenSep))
	}
	return false, "", ""
}

// peekIsEndOfToken
func (l *Lexer) peekIsEndOfToken() bool {
	return false
}

// resetAfterPeek will go back specified amount on the cursor
func (l *Lexer) resetAfterPeek(back int) {
	l.position = l.position - back
	l.readPosition = l.readPosition - back
}

// isText only deals with any text characters defined as
// outside of the capture group
func isText(ch byte) bool {
	return !nonText[string(ch)]
}

func newToken(tokenType config.TokenType, ch byte) config.Token {
	return config.Token{Type: tokenType, Literal: string(ch)}
}
