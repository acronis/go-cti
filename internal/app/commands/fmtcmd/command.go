package fmtcmd

import (
	"context"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
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

	return nil
}
