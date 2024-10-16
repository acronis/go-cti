package command

import (
	"github.com/acronis/go-cti/pkg/pacman"
	"github.com/acronis/go-cti/pkg/storage/gitstorage"
	"github.com/spf13/cobra"
)

func InitializePackageManager(_ *cobra.Command) (pacman.PackageManager, error) { // get option from command
	return pacman.New(
		pacman.WithStorage(gitstorage.New()),
	)
}
