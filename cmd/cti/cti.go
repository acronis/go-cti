package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/acronis/go-raml/stacktrace"

	"github.com/acronis/go-cti/internal/app/commands/buildcmd"
	"github.com/acronis/go-cti/internal/app/commands/depcmd"
	"github.com/acronis/go-cti/internal/app/commands/deploycmd"
	"github.com/acronis/go-cti/internal/app/commands/envcmd"
	"github.com/acronis/go-cti/internal/app/commands/fmtcmd"
	"github.com/acronis/go-cti/internal/app/commands/getcmd"
	"github.com/acronis/go-cti/internal/app/commands/infocmd"
	"github.com/acronis/go-cti/internal/app/commands/initcmd"
	"github.com/acronis/go-cti/internal/app/commands/lintcmd"
	"github.com/acronis/go-cti/internal/app/commands/restcmd"
	"github.com/acronis/go-cti/internal/app/commands/testcmd"
	"github.com/acronis/go-cti/internal/app/commands/validatecmd"
	"github.com/acronis/go-cti/internal/app/commands/versioncmd"

	"github.com/acronis/go-cti/internal/app/cti"
	"github.com/acronis/go-cti/internal/pkg/command"
	"github.com/acronis/go-cti/internal/pkg/execx"
	"github.com/acronis/go-cti/internal/pkg/slogex"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	slogformatter "github.com/samber/slog-formatter"
	"github.com/spf13/cobra"
)

type CommandError struct {
	Inner error
	Msg   string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("%s: %v", e.Msg, e.Inner)
}

func (e *CommandError) Unwrap() error {
	return e.Inner
}

func NewCommandError(err error, msg string) error {
	if err != nil {
		return &CommandError{Inner: err, Msg: msg}
	}
	return nil
}

func InitLoggingAndRun(ctx context.Context, verbosity int, cmd command.Command) error {
	logLvl := func() slog.Level {
		if verbosity > 0 {
			return slog.LevelDebug
		}

		return slog.LevelInfo
	}()
	w := os.Stderr
	logger := slog.New(
		slogformatter.NewFormatterHandler(
			slogformatter.HTTPRequestFormatter(false),
			slogformatter.HTTPResponseFormatter(false),
			slogformatter.FormatByType[[]string](func(s []string) slog.Value {
				return slog.StringValue(strings.Join(s, ","))
			}),
		)(
			tint.NewHandler(w, &tint.Options{
				Level:      logLvl,
				TimeFormat: time.TimeOnly,
				NoColor:    !isatty.IsTerminal(w.Fd()),
			}),
		),
	)
	slog.SetDefault(logger)

	return NewCommandError(cmd.Execute(ctx), "command error")
}

func main() {
	os.Exit(mainFn())
}

func mainFn() int {
	opts := cti.Options{}
	var verbosity int
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	cmdBuild := &cobra.Command{
		Use:   "pack",
		Short: "pack cti bundle",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, buildcmd.New(opts, args))
		},
	}

	cmdDep := &cobra.Command{
		Use:   "dep",
		Short: "tool to manage cti dependencies",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, depcmd.New(opts, args))
		},
	}

	cmdDeploy := &cobra.Command{
		Use:   "deploy",
		Short: "build and deploy cti package and dependencies to testing stand or production",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, deploycmd.New(opts, args))
		},
	}

	cmdEnv := &cobra.Command{
		Use:   "env",
		Short: "print cti environment information",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, envcmd.New(opts, args))
		},
	}

	cmdFmt := &cobra.Command{
		Use:   "fmt",
		Short: "cti fmt (reformat) cti sources",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, fmtcmd.New(opts, args))
		},
	}

	cmdGet := func() *cobra.Command {
		getOpts := getcmd.GetOptions{}
		cmd := &cobra.Command{
			Use:   "get",
			Short: "tool to download cti packages",
			Args:  cobra.MinimumNArgs(0),
			RunE: func(cmd *cobra.Command, args []string) error {

				return InitLoggingAndRun(ctx, verbosity, getcmd.New(opts, getOpts, args))
			},
		}

		cmd.Flags().BoolVarP(&getOpts.Replace, "replace", "r", false, "Replace package source on conflict.")

		return cmd
	}()

	cmdInfo := &cobra.Command{
		Use:   "info",
		Short: "print detailed information for cti package",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, infocmd.New(opts, args))
		},
	}

	cmdInit := &cobra.Command{
		Use:   "init",
		Short: "generate cti project with default dependencies",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, initcmd.New(opts, args))
		},
	}

	cmdLint := &cobra.Command{
		Use:   "lint",
		Short: "lint cti package",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, lintcmd.New(opts, args))
		},
	}

	cmdRest := &cobra.Command{
		Use:   "rest",
		Short: "run http server to expose restful api",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, restcmd.New(opts, args))
		},
	}

	cmdTest := &cobra.Command{
		Use:   "test",
		Short: "test cti package",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, testcmd.New(opts, args))
		},
	}

	cmdValidate := &cobra.Command{
		Use:   "validate",
		Short: "validate cti",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, validatecmd.New(opts, args))
		},
	}

	cmdVersion := &cobra.Command{
		Use:   "version",
		Short: "print a version of tool",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			return InitLoggingAndRun(ctx, verbosity, versioncmd.New(opts, args))
		},
	}

	rootCmd := func() *cobra.Command {
		cmd := &cobra.Command{
			Use:              "cti",
			Short:            "cti is a tool for managing cti projects",
			PersistentPreRun: func(cmd *cobra.Command, args []string) {},
			SilenceUsage:     true,
			SilenceErrors:    true,
			CompletionOptions: cobra.CompletionOptions{
				DisableDefaultCmd: true,
			},
		}

		cmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "Log with info log level.")

		cmd.AddCommand(
			cmdBuild,
			cmdDep,
			cmdDeploy,
			cmdEnv,
			cmdFmt,
			cmdGet,
			cmdInfo,
			cmdInit,
			cmdLint,
			cmdRest,
			cmdTest,
			cmdValidate,
			cmdVersion,
		)
		return cmd
	}()

	if err := rootCmd.Execute(); err != nil {
		var cmdErr *CommandError
		var execError *execx.ExecutionError
		if errors.As(err, &execError) {
			slog.Error(`                ^                   `)
			slog.Error(`              / | \                 `)
			slog.Error(`                |                   `)
			slog.Error(`                |                   `)
			slog.Error(` _____  ____   ____    ___   ____   `)
			slog.Error(`| ____||  _ \ |  _ \  / _ \ |  _ \  `)
			slog.Error(`|  _|  | |_) || |_) || | | || |_) | `)
			slog.Error(`| |___ |  _ < |  _ < | |_| ||  _ <  `)
			slog.Error(`|_____||_| \_\|_| \_\ \___/ |_| \_\ `)
			slog.Error(`                |                   `)
			slog.Error(`                |                   `)
			slog.Error(`                |                   `)
		}
		if errors.As(err, &cmdErr) {
			var st *stacktrace.StackTrace
			var isStackTrace bool
			if cmdErr.Inner != nil {
				st, isStackTrace = stacktrace.Unwrap(cmdErr.Inner)
			}
			if isStackTrace {
				slog.Error(fmt.Sprintf("Command failed: traceback:\n%s", st.Sprint()))
			} else {
				slog.Error("Command failed", slogex.Error(err))
			}
		} else {
			_ = rootCmd.Usage()
		}
		return 1
	}

	return 0
}
