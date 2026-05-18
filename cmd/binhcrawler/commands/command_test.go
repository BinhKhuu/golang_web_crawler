package commands

import (
	"bytes"
	"strings"
	"testing"
)

func Test_LogLevel_Empty_Default(t *testing.T) {
	buff := &bytes.Buffer{}

	cmd, err := SetupBaseCommand(buff, "")
	if err != nil {
		t.Errorf("Error setting up BaseCommand %v\n", err)
	}
	if cmd.Out != buff {
		t.Error("BaseCommand Out property not provided")
	}

	if cmd.Logger == nil {
		t.Error("BaseCommand Logger is nil")
	}
}

func Test_InvalildBuffer_ReturnError(t *testing.T) {
	_, err := SetupBaseCommand(nil, "")
	if err == nil {
		t.Error("Expected error when buffer is nil")
	}
}

func Test_LogLevel_Valid_SetsLevel_AndWritesToBuffer(t *testing.T) {
	tc := []struct {
		logLevel string
	}{
		{LogLevelDebug},
		{LogLevelInfo},
		{LogLevelWarn},
		{LogLevelError},
	}

	for _, test := range tc {
		t.Run(test.logLevel, func(t *testing.T) {
			buff := &bytes.Buffer{}
			cmd, err := SetupBaseCommand(buff, test.logLevel)
			if err != nil {
				t.Errorf("Error setting up BaseCommand %v\n", err)
			}
			if cmd.Logger == nil {
				t.Errorf("Logger return nil for %s", test.logLevel)
			}

			cmd.Logger.Debug("Debug Message")
			cmd.Logger.Info("Info Message")
			cmd.Logger.Warn("Warn Messag")
			cmd.Logger.Error("Error Message")

			output := buff.String()

			switch test.logLevel {
			case LogLevelError:
				if strings.Contains(output, "Debug Message") ||
					strings.Contains(output, "Info Message") ||
					strings.Contains(output, "Warn Messag") {
					t.Errorf("Expected to see error message only got %s", output)
				}
			case LogLevelWarn:
				if strings.Contains(output, "Debug Message") ||
					strings.Contains(output, "Info Message") {
					t.Errorf("Expected to see warn and error message only got %s", output)
				}
			case LogLevelInfo:
				if strings.Contains(output, "Debug Message") {
					t.Errorf("Expected to see info, warn and error message only got %s", output)
				}

			case LogLevelDebug:
				if !strings.Contains(output, "Debug Message") ||
					!strings.Contains(output, "Info Message") ||
					!strings.Contains(output, "Warn Messag") ||
					!strings.Contains(output, "Error Message") {
					t.Errorf("Expected to see debug, info, warn and error message only got %s", output)
				}
			}
		})
	}
}
