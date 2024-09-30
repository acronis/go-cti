package testsupp

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dusted-go/logging/prettylog"
	"github.com/stretchr/testify/require"
)

func ToRootDir(t *testing.T, relPath string) {
	slog.SetDefault(slog.New(
		prettylog.New(&slog.HandlerOptions{
			Level:       slog.LevelDebug,
			AddSource:   false,
			ReplaceAttr: nil,
		},
			prettylog.WithDestinationWriter(os.Stdout),
			func() prettylog.Option {
				return func(_ *prettylog.Handler) {}
			}(),
		)))

	_, b, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(b), relPath)
	require.NoError(t, os.Chdir(root))
}
