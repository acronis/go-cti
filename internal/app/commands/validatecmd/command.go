package validatecmd

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	"github.com/acronis/go-cti/pkg/depman"
	"github.com/acronis/go-cti/pkg/index"
	"github.com/acronis/go-cti/pkg/parser"
	"github.com/acronis/go-cti/pkg/validator"
)

type cmd struct {
	opts    cti.Options
	targets []string
}

func New(opts cti.Options, targets []string) command.Command {
	return &cmd{
		opts:    opts,
		targets: targets,
	}
}

func (c *cmd) Execute(ctx context.Context) error {
	// workDir := filepath.Dir(c.targets[0])
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	idxLockPath := filepath.Join(workDir, "index-lock.json")
	idxLock := depman.MakeIndexLock()
	if data, err := os.ReadFile(idxLockPath); err == nil {
		if err = json.Unmarshal(data, &idxLock); err != nil {
			return err
		}
	}

	p, err := parser.ParsePackage(filepath.Join(workDir, "index.json"))
	if err != nil {
		return err
	}
	if err = p.Serialize(); err != nil {
		return err
	}

	validator := validator.MakeCtiValidator()

	if err := validator.AddEntities(p.Registry.Total); err != nil {
		return err
	}
	// TODO: Move to PackageParser
	for _, dep := range idxLock.Packages {
		idx, err := index.ReadIndexFile(filepath.Join(workDir, ".dep", dep.AppCode, "index.json"))
		if err != nil {
			return err
		}
		for _, entityPath := range idx.Serialized {
			if err := validator.AddFromFile(filepath.Join(idx.BaseDir, entityPath)); err != nil {
				return err
			}
		}
	}
	// TODO: Validation for usage of indirect dependencies
	if errs := validator.ValidateAll(); errs != nil {
		for i := range errs {
			err := errs[i]
			slog.Error(err.Error())
		}
		return errors.New("failed to validate the package")
	}
	slog.Info("No errors found")

	return nil
}
