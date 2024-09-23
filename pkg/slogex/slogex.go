package slogex

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/acronis/go-raml/stacktrace"
)

func ErrorWithTrace(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}

	st, ok := stacktrace.Unwrap(err)
	if !ok {
		return slog.String("error", err.Error())
	}

	if st == nil {
		return slog.String("error", "nil stacktrace")
	}

	messageDelimiter := "\n"
	traceDelimiter := "\n\n\n"
	stackDelimiter := "\n\n"

	output := st.Sprint(stacktrace.WithEnsureDuplicates(), stacktrace.WithMessageDelimiter(messageDelimiter),
		stacktrace.WithTraceDelimiter(traceDelimiter), stacktrace.WithStackDelimiter(stackDelimiter))

	tracebacks := strings.Split(output, traceDelimiter)

	tracebackAttrs := make([]slog.Attr, 0, len(tracebacks))
	for traceIndex := range tracebacks {
		traceback := tracebacks[traceIndex]
		stacks := strings.Split(traceback, stackDelimiter)
		stackAttrs := make([]slog.Attr, 0, len(stacks))
		for stackIndex := range stacks {
			stack := stacks[stackIndex]
			header, message := strings.Split(stack, messageDelimiter)[0], strings.Split(stack, messageDelimiter)[1]
			key := fmt.Sprintf("%d", stackIndex)
			stackAttr := slog.Group(key, slog.String("header", header), slog.String("message", message))
			stackAttrs = append(stackAttrs, stackAttr)
		}
		tracebackAttr := slog.Group(fmt.Sprintf("%d", traceIndex), "stack", stackAttrs)
		tracebackAttrs = append(tracebackAttrs, tracebackAttr)
	}

	return slog.Group("tracebacks", "traces", tracebackAttrs)
}

func Error(err error) slog.Attr {
	return slog.Attr{Key: "error", Value: slog.StringValue(err.Error())}
}
