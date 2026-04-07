package ctipackage

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/acronis/go-cti/metadata"
	cmetadata "github.com/acronis/go-cti/metadata/collector/ctimetadata"
	cramlx "github.com/acronis/go-cti/metadata/collector/ramlx"
	"github.com/acronis/go-cti/metadata/registry"
	"github.com/acronis/go-cti/metadata/validator"
	"github.com/acronis/go-raml/v2"
)

func (pkg *Package) generateExamplesRAML() string {
	var sb strings.Builder
	sb.WriteString("#%RAML 1.0 Library\nuses:")
	for i, example := range pkg.Index.Examples {
		if strings.HasSuffix(example, RAMLExt) {
			sb.WriteString(fmt.Sprintf("\n  x%d: %s", i+1, example))
		}
	}
	return sb.String()
}

func (pkg *Package) parseExamplesRAML() (*registry.MetadataRegistry, error) {
	r, err := raml.ParseFromString(pkg.generateExamplesRAML(), "index_examples.raml", pkg.BaseDir, raml.OptWithValidate())
	if err != nil {
		return nil, fmt.Errorf("parse index_examples.raml: %w", err)
	}
	c, err := cramlx.NewRAMLXCollector(r)
	if err != nil {
		return nil, fmt.Errorf("create ramlx collector: %w", err)
	}
	return c.Collect()
}

func (pkg *Package) parseExamplesCTIMetadata() (*registry.MetadataRegistry, error) {
	fragments := make(map[string][]byte, len(pkg.Index.Examples))
	for _, example := range pkg.Index.Examples {
		if !strings.HasSuffix(example, YAMLExt) {
			continue
		}
		b, err := os.ReadFile(path.Join(pkg.BaseDir, example))
		if err != nil {
			return nil, fmt.Errorf("read example %s: %w", example, err)
		}
		fragments[example] = b
	}
	return cmetadata.NewCTIMetadataCollector(fragments, pkg.BaseDir).Collect()
}

// ParseExamples parses all example files listed in Index.Examples into ExamplesRegistry.
// It implicitly calls Parse if the package has not been parsed yet.
// Returns an error if any example CTI expression collides with an entity in GlobalRegistry.
func (pkg *Package) ParseExamples() error {
	if !pkg.Parsed {
		if err := pkg.Parse(); err != nil {
			return fmt.Errorf("parse package: %w", err)
		}
	}

	if len(pkg.Index.Examples) == 0 {
		return nil
	}

	examplesRAMLReg, err := pkg.parseExamplesRAML()
	if err != nil {
		return fmt.Errorf("parse examples RAML: %w", err)
	}

	examplesYAMLReg, err := pkg.parseExamplesCTIMetadata()
	if err != nil {
		return fmt.Errorf("parse examples CTI metadata: %w", err)
	}

	if err = examplesRAMLReg.CopyFrom(examplesYAMLReg); err != nil {
		return fmt.Errorf("merge examples registries: %w", err)
	}

	for ctiExpr := range examplesRAMLReg.Index {
		if _, exists := pkg.GlobalRegistry.Index[ctiExpr]; exists {
			return fmt.Errorf("example entity %q collides with an existing entity in the package", ctiExpr)
		}
	}

	pkg.ExamplesRegistry = examplesRAMLReg
	return nil
}

// ValidateExamples validates all example entities.
// It implicitly calls ParseExamples if the ExamplesRegistry has not been populated yet.
// Example entities are validated against a context that includes the package's GlobalRegistry
// and the ExamplesRegistry itself, so examples can reference each other.
// Optional ValidatorOptions (e.g. custom type/instance rules) are forwarded to the validator.
func (pkg *Package) ValidateExamples(opts ...validator.ValidatorOption) error {
	if pkg.ExamplesRegistry == nil {
		if err := pkg.ParseExamples(); err != nil {
			return fmt.Errorf("parse examples: %w", err)
		}
	}

	if pkg.ExamplesRegistry == nil || len(pkg.ExamplesRegistry.Index) == 0 {
		return nil
	}

	contextReg := registry.New()
	if err := contextReg.CopyFrom(pkg.GlobalRegistry); err != nil {
		return fmt.Errorf("copy global registry into context: %w", err)
	}
	if err := contextReg.CopyFrom(pkg.ExamplesRegistry); err != nil {
		return fmt.Errorf("copy examples registry into context: %w", err)
	}

	// Link example entities to their parent types. Parse() runs transformer.Transform()
	// on GlobalRegistry, but ExamplesRegistry is built afterwards, so parent pointers
	// on example entities are never set by the transformer. We set them here manually.
	for ctiExpr, entity := range pkg.ExamplesRegistry.Index {
		parentID := metadata.GetParentCTI(ctiExpr)
		if parentID == "" {
			continue
		}
		parent, ok := contextReg.Types[parentID]
		if !ok {
			return fmt.Errorf("parent type %s not found for example entity %s", parentID, ctiExpr)
		}
		if err := entity.SetParent(parent); err != nil {
			return fmt.Errorf("set parent for example entity %s: %w", ctiExpr, err)
		}
	}

	v, err := validator.New(pkg.Index.Vendor, pkg.Index.Pkg, contextReg, pkg.ExamplesRegistry, append(opts, validator.WithSkipOwnershipCheck())...)
	if err != nil {
		return fmt.Errorf("create validator: %w", err)
	}

	if pass, err := v.ValidateAll(); err != nil {
		if !pass {
			return fmt.Errorf("validate examples: %w", err)
		}
	}

	return nil
}
