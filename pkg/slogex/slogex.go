package slogex

import (
	"log/slog"

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

	var stackToGroup func(*stacktrace.StackTrace) slog.Attr
	stackToGroup = func(st *stacktrace.StackTrace) slog.Attr {
		if st == nil {
			return slog.Attr{}
		}

		frameGroup := slog.Group("frame", slog.String("location", st.Location), slog.String("message", st.FullMessageWithInfo()))

		if st.Wrapped != nil {
			return slog.Group("stack", frameGroup, stackToGroup(st.Wrapped))
		}

		return frameGroup
	}

	return slog.Group("tracebacks", stackToGroup(st))
}

func Error(err error) slog.Attr {
	return slog.Attr{Key: "error", Value: slog.StringValue(err.Error())}
}
