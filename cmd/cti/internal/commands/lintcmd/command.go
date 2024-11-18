package lintcmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
)

func New(_ context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "lint",
		Short: "lint cti package",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
}
