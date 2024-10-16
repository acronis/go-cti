package packcmd

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/pkg/archiver/tgzwriter"
	"github.com/acronis/go-cti/pkg/archiver/zippacker"
	"github.com/acronis/go-cti/pkg/ctipackage"
	"github.com/acronis/go-cti/pkg/packer"
	"github.com/spf13/cobra"
)

type PackOptions struct {
	FileName      string
	Prefix        string
	IncludeSource bool
	Format        PackFormat
}

func New(ctx context.Context) *cobra.Command {
	packOpts := PackOptions{}
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "pack cti package",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := command.GetWorkingDir(cmd)
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			return command.WrapError(execute(ctx, baseDir, packOpts))
		},
	}

	cmd.Flags().StringVarP(&packOpts.FileName, "output", "o", "package."+packer.ArchiveExtension, "Output file name with path.")
	cmd.Flags().StringVarP(&packOpts.Prefix, "prefix", "p", "", "Output prefix.")
	cmd.Flags().BoolVarP(&packOpts.IncludeSource, "include-source", "s", false, "Include source files in the resulting package.")
	cmd.Flags().Var(&packOpts.Format, "format", `Archive format. allowed: `+strings.Join(ListPackFormats, ","))

	return cmd
}

func execute(_ context.Context, baseDir string, opts PackOptions) error {
	slog.Info("Packing package", slog.String("path", baseDir))

	prkOpts := []packer.Option{}

	switch opts.Format {
	case PackFormatZip:
		prkOpts = append(prkOpts, packer.WithArchiver(zippacker.New()))
	case PackFormatTgz:
		fallthrough
	default:
		prkOpts = append(prkOpts, packer.WithArchiver(tgzwriter.New()))
	}

	if opts.IncludeSource {
		prkOpts = append(prkOpts, packer.WithSources())
	}
	p, err := packer.New(prkOpts...)
	if err != nil {
		return fmt.Errorf("new packer: %w", err)
	}

	pkg, err := ctipackage.New(baseDir)
	if err != nil {
		return fmt.Errorf("new package: %w", err)
	}

	fullPath := filepath.Join(opts.Prefix, opts.FileName)

	if err := p.Pack(pkg, fullPath); err != nil {
		return fmt.Errorf("pack the package: %w", err)
	}

	slog.Info("Packing has been completed", "path", fullPath)
	return nil
}
