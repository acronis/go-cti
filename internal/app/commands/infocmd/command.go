package infocmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
)

func New(_ context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "print detailed information for cti package",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
}
