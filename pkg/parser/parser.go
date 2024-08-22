package parser

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/index"
	"github.com/acronis/go-cti/pkg/ramlx"
)

type Parser struct {
	Path  string
	RamlX *ramlx.RamlX
}

func NewRamlParser(path string) (*Parser, error) {
	p, err := ramlx.NewRamlX()
	if err != nil {
		return nil, err
	}

	return &Parser{
		Path:  path,
		RamlX: p,
	}, nil
}

func (p *Parser) GetPackageDir() string {
	return filepath.Dir(p.Path)
}

// ParseAll cti type, instances, dictionaries in one single output
// Parser will take a path for example "/home/app-package/index.json".
func (p *Parser) ParseAll() (CtiEntities, error) {
	out, err := p.RamlX.ParseIndexFile(p.Path)
	if err != nil {
		return nil, err
	}

	output, err := ValidateParserOutput(out)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// Parse parses a single entity file
// Parser will take a path for example "/home/app-package/test.raml".
func (p *Parser) Parse(path string) (CtiEntities, error) {
	out, err := p.RamlX.ParseEntityFile(path)
	if err != nil {
		return nil, err
	}

	output, err := ValidateParserOutput(out)
	if err != nil {
		return nil, err
	}

	// utils.ValidateParserOutput returns a slice of outputs to accommodate parsing in ParseAll().
	// in our case, it should always return a single element within the slice.
	if len(output) != 1 {
		return nil, errors.New("error parsing entity file: invalid number of outputs")
	}

	return output, nil
}

func (p *Parser) Bundle(destination string) error {
	entities, err := p.ParseAll()
	if err != nil {
		return err
	}

	idx, err := index.OpenIndexFile(p.Path)
	if err != nil {
		return err
	}

	assets := make(map[string]string)
	for _, path := range idx.Data.Assets {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		assets[path] = base64.StdEncoding.EncodeToString(bytes)
	}

	bundle := Bundle{
		Entities: entities,
		Assets:   assets,
	}
	bytes, err := json.Marshal(bundle)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destination, "bundle.cti"), bytes, 0o644)
}

// ValidateParserOutput to validate output from Parser.
func ValidateParserOutput(out []byte) (CtiEntities, error) {
	var parserOutputs CtiEntities
	var parserErrors ParserErrors

	err := json.Unmarshal(out, &parserErrors)
	if err == nil {
		return nil, fmt.Errorf("invalid errors output: %v", string(out))
	}

	err = json.Unmarshal(out, &parserOutputs)
	if err != nil {
		return nil, fmt.Errorf("invalid entities output: %v", string(out))
	}

	return parserOutputs, nil
}
