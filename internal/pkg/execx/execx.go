package execx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Command defines a command specification.
type Command struct {
	Args      []string
	ExtraVars []string
	Dir       string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}

// ExecString executes command and returns its stdout as string.
func ExecString(ctx context.Context, args []string) (string, error) {
	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	if err := Exec(ctx, Command{Args: args, Stdout: outBuf, Stderr: errBuf}); err != nil {
		if errBuf.Len() != 0 {
			err = fmt.Errorf("command %q stderr %q: %w", args, errBuf.String(), err)
		} else {
			err = fmt.Errorf("command %q: %w", args, err)
		}

		return "", err
	}

	return strings.TrimSpace(outBuf.String()), nil
}

// Exec executes command.
func Exec(ctx context.Context, opts Command) error {
	slog.Info("exec",
		slog.Any("cmd", opts.Args),
		slog.Any("extra-vars", opts.ExtraVars),
		slog.String("cwd", opts.Dir))

	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, opts.Args[0], opts.Args[1:]...)
	cmd.Env = append(os.Environ(), opts.ExtraVars...)
	cmd.Dir = opts.Dir

	debugLogLevel := slog.Default().Enabled(ctx, slog.LevelDebug)
	echoStdout := debugLogLevel
	echoStderr := debugLogLevel

	var cmdStdin io.WriteCloser
	var cmdStdout, cmdStderr io.ReadCloser
	defer func() {
		// TODO ensure closed
		if cmdStdin != nil {
			cmdStdin.Close()
		}
		if cmdStdout != nil {
			cmdStdout.Close()
		}
		if cmdStderr != nil {
			cmdStderr.Close()
		}
	}()

	var wg sync.WaitGroup

	// redirect or pipe stdout
	if opts.Stdout != nil {
		stdout := opts.Stdout
		if echoStdout {
			stdout = io.MultiWriter(os.Stdout, opts.Stdout)
		}
		var err error
		if cmdStdout, err = cmd.StdoutPipe(); err != nil {
			return NewExecutionError(err, "stdout pipe")
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(stdout, cmdStdout); err != nil {
				fmt.Printf("error copying to stdout: %q", err)
			}
		}()
	} else if echoStdout {
		cmd.Stdout = os.Stdout
	}

	// redirect stdin
	if opts.Stderr != nil {
		stderr := opts.Stderr
		if echoStdout {
			stderr = io.MultiWriter(os.Stderr, opts.Stderr)
		}
		var err error
		if cmdStderr, err = cmd.StderrPipe(); err != nil {
			return NewExecutionError(err, "stderr pipe")
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(stderr, cmdStderr); err != nil {
				fmt.Println(err)
			}
		}()
	} else if echoStderr {
		cmd.Stderr = os.Stderr
	}

	if opts.Stdin != nil {
		var err error
		if cmdStdin, err = cmd.StdinPipe(); err != nil {
			return NewExecutionError(err, "stdin pipe")
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(cmdStdin, opts.Stdin); err != nil {
				fmt.Println(err)
			}
		}()
	}

	fmt.Println()
	// start process
	if err := cmd.Start(); err != nil {
		return NewExecutionError(err, "exec")
	}

	wg.Wait()

	defer fmt.Println()

	if err := cmd.Wait(); err != nil {
		return NewExecutionError(err, "wait")
	}

	return nil
}
