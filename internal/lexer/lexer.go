// Package lexer
//
// Performs lexical analysis on the source files and emits tokens.
package lexer

import (
	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/token"
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
	// AWS|AZure
	"A": true,
	// VAULT (HashiCorp)
	"V": true,
	// GCP
	"G": true,
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
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	switch l.ch {
	// identify the dynamically selected key separator
	case l.keySeparator:
		tok = token.Token{Type: token.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR, Literal: string(l.ch)}
	// Specific cases for BEGIN_CONFIGMANAGER_TOKEN possibilities
	case 'A':
		if l.peekChar() == 'W' {
			// AWS store types
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.SecretMgrPrefix, config.ParamStorePrefix}, "AW"); found {
				tok = token.Token{Type: token.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker AW as text
				tok = token.Token{Type: token.TEXT, Literal: "AW"}
			}
		} else if l.peekChar() == 'Z' {
			// Azure Store Types
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.AzKeyVaultSecretsPrefix, config.AzTableStorePrefix, config.AzAppConfigPrefix}, "AZ"); found {
				tok = token.Token{Type: token.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker AZ as text
				tok = token.Token{Type: token.TEXT, Literal: "AZ"}
			}
		} else {
			tok = token.Token{Type: token.TEXT, Literal: "A"}
		}
	case 'G':
		// GCP TOKENS
		if l.peekChar() == 'C' {
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.GcpSecretsPrefix}, "GC"); found {
				tok = token.Token{Type: token.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker - GC literal as text
				tok = token.Token{Type: token.TEXT, Literal: "GC"}
			}
		} else {
			tok = token.Token{Type: token.TEXT, Literal: "G"}
		}
	case 'V':
		// HASHI VAULT Tokens
		if l.peekChar() == 'A' {
			l.readChar()
			if found, literal, imp := l.peekIsBeginOfToken([]config.ImplementationPrefix{config.HashicorpVaultPrefix}, "VA"); found {
				tok = token.Token{Type: token.BEGIN_CONFIGMANAGER_TOKEN, Literal: literal, ImpPrefix: imp}
			} else {
				// it is not a marker VA as text
				tok = token.Token{Type: token.TEXT, Literal: "VA"}
			}
		} else {
			tok = token.Token{Type: token.TEXT, Literal: "V"}
		}
	case '=':
		tok = token.Token{Type: token.EQUALS, Literal: "="}
	case '.':
		tok = token.Token{Type: token.DOT, Literal: "."}
	case ',':
		tok = token.Token{Type: token.COMMA, Literal: ","}
	case '/':
		if l.peekChar() == '?' {
			l.readChar()
			tok = token.Token{Type: token.SLASH_QUESTION_MARK, Literal: "/?"}
		} else {
			tok = token.Token{Type: token.FORWARD_SLASH, Literal: "/"}
		}
	case '\\':
		tok = token.Token{Type: token.BACK_SLASH, Literal: "\\"}
	case '?':
		tok = token.Token{Type: token.QUESTION_MARK, Literal: "?"}
	case ']':
		tok = token.Token{Type: token.END_META_CONFIGMANAGER_TOKEN, Literal: "]"}
	case '[':
		tok = token.Token{Type: token.BEGIN_META_CONFIGMANAGER_TOKEN, Literal: "["}
	case '|':
		tok = token.Token{Type: token.PIPE, Literal: "|"}
	case '@':
		tok = token.Token{Type: token.AT_SIGN, Literal: "@"}
	case ':':
		tok = token.Token{Type: token.COLON, Literal: ":"}
	case '"':
		tok = token.Token{Type: token.DOUBLE_QUOTE, Literal: "\""}
	case '\'':
		tok = token.Token{Type: token.SINGLE_QUOTE, Literal: "'"}
	case '\n':
		l.line = l.line + 1
		l.column = 0 // reset column count
		tok = l.setTextSeparatorToken()
		// want to preserve all indentations and punctuation
	case ' ', '\r', '\t', '\f':
		tok = l.setTextSeparatorToken()
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isText(l.ch) {
			tok.Literal = l.readText()
			tok.Type = token.TEXT
			return tok
		}
		tok = newToken(token.ILLEGAL, l.ch)
	}
	// add general properties to each token
	tok.Line = l.line
	tok.Column = l.column
	tok.Source = token.Source{Path: l.source.FullPath, File: l.source.FileName}
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

func (l *Lexer) setTextSeparatorToken() token.Token {
	tok := newToken(token.LookupIdent(string(l.ch)), l.ch)
	return tok
}

// peekIsBeginOfToken attempts to identify the possible token
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

func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}
