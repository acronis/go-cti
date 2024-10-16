package pkgcmd

import (
	"context"

	"github.com/acronis/go-cti/internal/app/commands/pkgcmd/downloadcmd"
	"github.com/acronis/go-cti/internal/app/commands/pkgcmd/getcmd"

	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "command to manage cti packages",
	}
	cmd.AddCommand(
		getcmd.New(ctx),
		downloadcmd.New(ctx),
	)
	return cmd
}
