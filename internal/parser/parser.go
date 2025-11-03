package parser

import (
	"errors"
	"fmt"
	"os"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/lexer"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
)

func wrapErr(incompleteToken string, line, position int, etyp error) error {
	return fmt.Errorf("\n- token: (%s) on line: %d column: %d] %w", incompleteToken, line, position, etyp)
}

var (
	ErrNoEndTagFound                 = errors.New("no corresponding end tag found")
	ErrUnableToReplaceVarPlaceholder = errors.New("variable specified in the content was not found in the environment")
)

type ConfigManagerTokenBlock struct {
	BeginToken  config.Token
	ParsedToken config.ParsedTokenConfig
	EndToken    config.Token
}

type Parser struct {
	l         *lexer.Lexer
	errors    []error
	log       log.ILogger
	curToken  config.Token
	peekToken config.Token
	config    *config.GenVarsConfig
	environ   []string
}

func New(l *lexer.Lexer, c *config.GenVarsConfig) *Parser {
	p := &Parser{
		l:       l,
		log:     log.New(os.Stderr),
		errors:  []error{},
		config:  c,
		environ: os.Environ(),
	}

	// Read two tokens, so curToken and peekToken are both set
	// first one sets the curToken to the value of peekToken -
	// which at this point is just the first upcoming token
	p.nextToken()
	// second one sets the curToken to the actual value of the first upcoming
	// token and peekToken is the actual second upcoming token
	p.nextToken()

	return p
}

func (p *Parser) WithEnvironment(environ []string) *Parser {
	p.environ = environ
	return p
}

func (p *Parser) WithLogger(logger log.ILogger) *Parser {
	p.log = nil //speed up GC
	p.log = logger
	return p
}

// Parse creates a flat list of ConfigManagerTokenBlock
// In the order they were declared in the source text
//
// The parser does not do a second pass and interprets the source from top to bottom
func (p *Parser) Parse() ([]ConfigManagerTokenBlock, []error) {
	stmts := []ConfigManagerTokenBlock{}

	for !p.currentTokenIs(config.EOF) {
		if p.currentTokenIs(config.BEGIN_CONFIGMANAGER_TOKEN) {
			// continues to read the tokens until it hits an end token or errors
			configManagerToken, err := config.NewToken(p.curToken.ImpPrefix, *p.config)
			if err != nil {
				return nil, []error{err}
			}
			if stmt := p.buildConfigManagerTokenFromBlocks(configManagerToken); stmt != nil {
				stmts = append(stmts, *stmt)
			}
		}
		p.nextToken()
	}

	return stmts, p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) currentTokenIs(t config.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t config.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) peekTokenIsEnd() bool {
	endTokens := map[config.TokenType]bool{
		config.AT_SIGN: true, config.QUESTION_MARK: true, config.COLON: true,
		config.SLASH_QUESTION_MARK: true, config.EOF: true,
		// traditional ends of tokens
		config.DOUBLE_QUOTE: true, config.SINGLE_QUOTE: true, config.SPACE: true,
		config.NEW_LINE: true,
	}
	return endTokens[p.peekToken.Type]
}

// buildConfigManagerTokenFromBlocks
func (p *Parser) buildConfigManagerTokenFromBlocks(configManagerToken *config.ParsedTokenConfig) *ConfigManagerTokenBlock {
	currentToken := p.curToken
	stmt := &ConfigManagerTokenBlock{BeginToken: currentToken}

	// move past current token
	p.nextToken()

	fullToken := currentToken.Literal
	// built as part of the below parser
	sanitizedToken := ""

	// should exit the loop if no end tag found
	notFoundEnd := true

	// stop on end of file
	for !p.peekTokenIs(config.EOF) {
		// // This is the target state when there is an optional token wrapping
		// // 	e.g. `{{ IMP://path }}`
		// // currently this is untestable
		// if p.peekTokenIs(config.END_CONFIGMANAGER_TOKEN) {
		// 	notFoundEnd = false
		// 	fullToken += p.curToken.Literal
		// 	sanitizedToken += p.curToken.Literal
		// 	stmt.EndToken = p.curToken
		// 	break
		// }

		// when next token is another token
		// i.e. the tokens are adjacent
		if p.peekTokenIs(config.BEGIN_CONFIGMANAGER_TOKEN) {
			notFoundEnd = false
			fullToken += p.curToken.Literal
			sanitizedToken += p.curToken.Literal
			stmt.EndToken = p.curToken
			break
		}

		// reached the end of token
		if p.peekTokenIsEnd() {
			// we want set the current token as both the full and sanitized
			// the current lexer token is the entire configmanager token
			notFoundEnd = false
			fullToken += p.curToken.Literal
			sanitizedToken += p.curToken.Literal
			stmt.EndToken = p.curToken
			break
		}

		//sample token will be consumed like this
		// AWSSECRETS:///path/to/my/key|lookup.Inside.Object[meta=data]
		//
		// everything is token path until (if any key separator exists)
		// check key separator this marks the end of a normal token path
		//
		// keyLookup and Metadata are optional - is always specified in that order
		if p.currentTokenIs(config.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR) {
			if err := p.buildKeyPathSeparator(configManagerToken); err != nil {
				p.errors = append(p.errors, wrapErr(fullToken, currentToken.Line, currentToken.Column, err))
				return nil
			}
			notFoundEnd = false
			// keyPath would have built the keyPath and metadata if any
			break
		}

		// optionally at the end of the path without key separator
		// check metadata there can be a metadata bracket `[key=val,k1=v2]`
		if p.currentTokenIs(config.BEGIN_META_CONFIGMANAGER_TOKEN) {
			if err := p.buildMetadata(configManagerToken); err != nil {
				p.errors = append(p.errors, wrapErr(fullToken, currentToken.Line, currentToken.Column, err))
				return nil
			}
			notFoundEnd = false
			break
		}

		sanitizedToken += p.curToken.Literal
		fullToken += p.curToken.Literal

		// when the next token is EOF
		// we want set the current token
		// else it would be lost once the parser is advanced below
		p.nextToken()
		if p.peekTokenIs(config.EOF) {
			notFoundEnd = false
			fullToken += p.curToken.Literal
			sanitizedToken += p.curToken.Literal
			stmt.EndToken = p.curToken
			break
		}
	}

	if notFoundEnd {
		p.errors = append(p.errors, wrapErr(fullToken, currentToken.Line, currentToken.Column, ErrNoEndTagFound))
		return nil
	}

	configManagerToken.WithSanitizedToken(sanitizedToken)
	stmt.ParsedToken = *configManagerToken

	return stmt
}

// buildKeyPathSeparator already advanced to the first token
func (p *Parser) buildKeyPathSeparator(configManagerToken *config.ParsedTokenConfig) error {
	// advance to next token i.e. post the path separator
	p.nextToken()
	keyPath := ""
	if p.peekTokenIs(config.EOF) {
		// if the next token EOF we set the path as current token and exit
		// otherwise we would never hit the below loop
		configManagerToken.WithKeyPath(p.curToken.Literal)
		return nil
	}
	for !p.peekTokenIs(config.EOF) {
		if p.peekTokenIs(config.BEGIN_META_CONFIGMANAGER_TOKEN) {
			// add current token to the keysPath and move onto the metadata
			keyPath += p.curToken.Literal
			p.nextToken()
			if err := p.buildMetadata(configManagerToken); err != nil {
				return err
			}
			break
		}
		// touching another token or end of token
		if p.peekTokenIs(config.BEGIN_CONFIGMANAGER_TOKEN) || p.peekTokenIsEnd() {
			keyPath += p.curToken.Literal
			break
		}
		keyPath += p.curToken.Literal
		p.nextToken()
		if p.peekTokenIs(config.EOF) {
			// check if the next token is EOF once advanced
			// if it is we want to consume current token else it will be skipped
			keyPath += p.curToken.Literal
			break
		}
	}
	configManagerToken.WithKeyPath(keyPath)
	return nil
}

var ErrMetadataEmpty = errors.New("emtpy metadata")

// buildMetadata adds metadata to the ParsedTokenConfig
func (p *Parser) buildMetadata(configManagerToken *config.ParsedTokenConfig) error {
	metadata := ""
	found := false
	if p.peekTokenIs(config.END_META_CONFIGMANAGER_TOKEN) {
		return fmt.Errorf("%w, metadata brackets must include at least one set of key=value pairs", ErrMetadataEmpty)
	}
	p.nextToken()
	for !p.peekTokenIs(config.EOF) {
		if p.peekTokenIsEnd() {
			if p.currentTokenIs(config.END_META_CONFIGMANAGER_TOKEN) {
				metadata += p.curToken.Literal
				found = true
				p.nextToken()
				break
			}
			return fmt.Errorf("%w, metadata string has no closing", ErrNoEndTagFound)
		}
		if p.peekTokenIs(config.END_META_CONFIGMANAGER_TOKEN) {
			metadata += p.curToken.Literal
			found = true
			p.nextToken()
			break
		}
		metadata += p.curToken.Literal
		p.nextToken()
	}
	configManagerToken.WithMetadata(metadata)

	if !found {
		return fmt.Errorf("%w, metadata string has no closing", ErrNoEndTagFound)
	}
	return nil
}
