package cmdutils

import (
	"fmt"
	"io"
	"strings"

	"github.com/dnitsch/configmanager/internal/config"
	"github.com/dnitsch/configmanager/pkg/generator"
)

// PostProcessor
// processes the rawMap and outputs the result
// depending on cmdline options
type PostProcessor struct {
	ProcessedMap generator.ParsedMap
	Config       *config.GenVarsConfig
	outString    []string
}

// ConvertToExportVar assigns the k/v out
// as unix style export key=val pairs separated by `\n`
func (p *PostProcessor) ConvertToExportVar(rawMap generator.ParsedMap) []string {
	for k, v := range rawMap {
		rawKeyToken := strings.Split(k, "/") // assumes a path like token was used
		topLevelKey := rawKeyToken[len(rawKeyToken)-1]
		trm := make(generator.ParsedMap)
		if parsedOk := generator.IsParsed(v, &trm); parsedOk {
			// if is a map
			// try look up on key if separator defined
			normMap := p.envVarNormalize(trm)
			p.exportVars(normMap)
			continue
		}
		p.exportVars(generator.ParsedMap{topLevelKey: v})
	}
	return p.outString
}

// envVarNormalize
func (p *PostProcessor) envVarNormalize(pmap generator.ParsedMap) generator.ParsedMap {
	normalizedMap := make(generator.ParsedMap)
	for k, v := range pmap {
		normalizedMap[p.normalizeKey(k)] = v
	}
	return normalizedMap
}

func (p *PostProcessor) exportVars(exportMap generator.ParsedMap) {

	for k, v := range exportMap {
		// NOTE: \n line ending is not totally cross platform
		t := fmt.Sprintf("%T", v)
		switch t {
		case "string":
			p.outString = append(p.outString, fmt.Sprintf("export %s='%s'", p.normalizeKey(k), v))
		default:
			p.outString = append(p.outString, fmt.Sprintf("export %s=%v", p.normalizeKey(k), v))
		}
	}
}

// normalizeKeys returns env var compatible key
func (p *PostProcessor) normalizeKey(k string) string {
	// the order of replacer pairs matters less
	// as the Replace builds a node tree without overlapping matches
	replacer := strings.NewReplacer([]string{" ", "", "@", "", "!", "", "-", "_", p.Config.KeySeparator(), "__"}...)
	return strings.ToUpper(replacer.Replace(k))
}

// FlushToFile saves contents to file provided
// in the config input into the generator
// default location is ./app.env
func (p *PostProcessor) FlushToFile(w io.Writer, out []string) error {
	return p.flushToFile(w, listToString(p.outString))
}

// StrToFile writes a provided string to the writer
func (p *PostProcessor) StrToFile(w io.Writer, str string) error {
	return p.flushToFile(w, str)
}

func (p *PostProcessor) flushToFile(f io.Writer, str string) error {

	_, e := f.Write([]byte(str))

	if e != nil {
		return e
	}

	return nil
}

func listToString(strList []string) string {
	return strings.Join(strList, "\n")
}
