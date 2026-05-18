package commands

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
)

// GlobalOpts holds options shared across all commands.
type GlobalOpts struct {
	LogLevel string `default:"info"                                       description:"Log level (debug, info, warn, error)" long:"log-level" short:"l"`
	DryRun   bool   `description:"Validate configuration without running" long:"dry-run"                                     short:"d"`
}

type BaseCommand struct {
	Out    io.Writer
	Logger *slog.Logger
}

// SetupBaseCommand initializes and returns a new BaseCommand with the provided buffer.
func SetupBaseCommand(buff *bytes.Buffer, logLevel string) (*BaseCommand, error) {
	if buff == nil {
		return nil, errors.New("buffer cannot be nil")
	}

	logger := slog.LevelInfo
	if logLevel != "" {
		switch logLevel {
		case LogLevelDebug:
			logger = slog.LevelDebug
		case LogLevelInfo:
			logger = slog.LevelInfo
		case LogLevelWarn:
			logger = slog.LevelWarn
		case LogLevelError:
			logger = slog.LevelError
		default:
			logger = slog.LevelInfo
		}
	}
	return &BaseCommand{
		Out:    buff,
		Logger: slog.New(slog.NewTextHandler(buff, &slog.HandlerOptions{Level: logger})),
	}, nil
}
