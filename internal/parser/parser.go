package parser

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/lexer"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"

	"github.com/a8m/envsubst"
)

func wrapErr(file string, line, position int, etyp error) error {
	return fmt.Errorf("\n - [%s:%d:%d] %w", file, line, position, etyp)
}

var (
	ErrNoEndTagFound                 = errors.New("no corresponding end tag found")
	ErrUnableToReplaceVarPlaceholder = errors.New("variable specified in the content was not found in the environment")
)

type ConfigManagerTokenBlock struct {
	BeginToken  config.Token
	ParsedToken config.ParsedTokenConfig
	Value       string
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
	genDocStms := []ConfigManagerTokenBlock{}

	for !p.currentTokenIs(config.EOF) {
		if p.currentTokenIs(config.BEGIN_CONFIGMANAGER_TOKEN) {
			// parseGenDocBlocks will advance the token until
			// it hits the END_DOC_GEN token
			configManagerToken, err := config.NewTokenConfig(p.curToken.ImpPrefix, *p.config)
			if err != nil {
				return nil, []error{err}
			}
			if stmt := p.buildConfigManagerTokenFromBlocks(configManagerToken); stmt != nil {
				genDocStms = append(genDocStms, *stmt)
			}
		}
		p.nextToken()
	}

	return genDocStms, p.errors
}

// ExpandEnvVariables expands the env vars inside DocContent
// to their environment var values.
//
// Failing when a variable is either not set or set but empty.
func ExpandEnvVariables(input string, vars []string) (string, error) {
	for _, v := range vars {
		kv := strings.Split(v, "=")
		key, value := kv[0], kv[1] // kv[1] will be an empty string = ""
		os.Setenv(key, value)
	}

	return envsubst.StringRestrictedNoDigit(input, true, true, false)
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
		config.DOUBLE_QUOTE: true, config.SINGLE_QUOTE: true,
		config.NEW_LINE: true,
	}
	return endTokens[p.peekToken.Type]
}

// buildConfigManagerTokenFromBlocks throws away all other content other
// than what is inside //+gendoc tags
// parses any annotation and creates GenDocBlock
// for later analysis
func (p *Parser) buildConfigManagerTokenFromBlocks(configManagerToken *config.ParsedTokenConfig) *ConfigManagerTokenBlock {
	currentToken := p.curToken
	stmt := &ConfigManagerTokenBlock{BeginToken: currentToken}

	// move past current token
	p.nextToken()

	fullToken := currentToken.Literal
	// pathLookup := ""
	// metadataPortion := ""

	// should exit the loop if no end doc tag found
	notFoundEnd := true
	// stop on end of file
	for !p.peekTokenIs(config.EOF) {

		// when next token is another token
		// i.e. the tokens are adjacent
		if p.peekTokenIs(config.BEGIN_CONFIGMANAGER_TOKEN) {
			notFoundEnd = false
			fullToken += p.curToken.Literal
			stmt.EndToken = p.curToken
			p.nextToken()
			break
		}

		// reached the end of a potential token
		if p.peekTokenIsEnd() {
			notFoundEnd = false
			fullToken += p.curToken.Literal
			stmt.EndToken = p.curToken
			break
		}

		//sample token will be consumed like this
		// AWSSECRETS:///path/to/my/key|lookup.Inside.Object[meta=data]
		//
		// everything is token path until (if any key separator exists)
		// check key separator this marks the end of a normal token path
		if p.currentTokenIs(config.CONFIGMANAGER_TOKEN_KEY_PATH_SEPARATOR) {
			// advance to next token i.e. start of the path separator
			p.nextToken()
			if err := p.buildKeyPathSeparator(configManagerToken); err != nil {
				p.errors = append(p.errors, err)
				return nil
			}
			p.nextToken()
			continue
		}
		// optionally at the end of the path (i.e. a JSONPath look up)
		// check metadata there can be a metadata bracket `[key=val,k1=v2]`
		if p.currentTokenIs(config.BEGIN_META_CONFIGMANAGER_TOKEN) {
			if err := p.buildMetadata(configManagerToken); err != nil {
				p.errors = append(p.errors, err)
				return nil
			}
		}
		fullToken += p.curToken.Literal
		p.nextToken()
	}

	if notFoundEnd {
		p.errors = append(p.errors, wrapErr(currentToken.Source.File, currentToken.Line, currentToken.Column, ErrNoEndTagFound))
		return nil
	}

	stmt.ParsedToken = *configManagerToken
	stmt.Value = fullToken

	return stmt
}

// buildKeyPathSeparator already advanced to the first token
func (p *Parser) buildKeyPathSeparator(configManagerToken *config.ParsedTokenConfig) error {
	keyPath := ""
	for !p.peekTokenIs(config.EOF) {
		if p.peekTokenIs(config.BEGIN_META_CONFIGMANAGER_TOKEN) {
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
	}
	configManagerToken.WithKeyPath(keyPath)
	return nil
}

// buildMetadata adds metadata to the ParsedTokenConfig
func (p *Parser) buildMetadata(configManagerToken *config.ParsedTokenConfig) error {
	// inLoop := true
	// errNoClose := fmt.Errorf("%w, metadata string has no closing", ErrNoEndTagFound)
	metadata := ""
	found := false
	for !p.peekTokenIs(config.EOF) {
		if p.peekTokenIsEnd() {
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
