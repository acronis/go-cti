package deploycmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
)

func New(_ context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "deploy",
		Short: "build and deploy cti package and dependencies to testing stand or production",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
}
