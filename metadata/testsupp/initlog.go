package testsupp

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/dusted-go/logging/prettylog"
	slogformatter "github.com/samber/slog-formatter"
)

func InitLog(t *testing.T) {
	t.Helper()

	funcHandler := slogformatter.NewFormatterHandler(
		slogformatter.FormatByType(func(s []string) slog.Value {
			return slog.StringValue(strings.Join(s, ","))
		}),
	)

	plHandler := prettylog.New(
		&slog.HandlerOptions{
			Level:       slog.LevelDebug,
			AddSource:   false,
			ReplaceAttr: nil,
		},
		prettylog.WithDestinationWriter(os.Stdout),
	)

	formatHandler := funcHandler(plHandler)

	logger := slog.New(formatHandler)
	slog.SetDefault(logger)
}
