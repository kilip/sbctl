package logger

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestNewSlogLogger(t *testing.T) {
	// Verify that NewSlogLogger doesn't crash and returns a valid logger.
	l := NewSlogLogger("info")
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestSlogLogger_LevelsAndOutputs(t *testing.T) {
	tests := []struct {
		name       string
		level      string
		logFunc    func(Logger)
		expected   string
		unexpected string
	}{
		{
			name:  "Debug level enables debug logs",
			level: "debug",
			logFunc: func(l Logger) {
				l.Debug("debug message", "key", "val")
			},
			expected: "debug message",
		},
		{
			name:  "Info level disables debug logs",
			level: "info",
			logFunc: func(l Logger) {
				l.Debug("debug message")
				l.Info("info message")
			},
			expected:   "info message",
			unexpected: "debug message",
		},
		{
			name:  "Warn level enables warn logs",
			level: "warn",
			logFunc: func(l Logger) {
				l.Warn("warn message")
			},
			expected: "warn message",
		},
		{
			name:  "Error level enables error logs",
			level: "error",
			logFunc: func(l Logger) {
				l.Error("error message")
			},
			expected: "error message",
		},
		{
			name:  "Default level is Info",
			level: "invalid-level",
			logFunc: func(l Logger) {
				l.Info("info default message")
			},
			expected: "info default message",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewSlogLoggerWithWriter(tc.level, &buf)
			tc.logFunc(l)

			output := buf.String()
			if tc.expected != "" && !strings.Contains(output, tc.expected) {
				t.Errorf("expected output to contain %q, got: %q", tc.expected, output)
			}
			if tc.unexpected != "" && strings.Contains(output, tc.unexpected) {
				t.Errorf("expected output NOT to contain %q, got: %q", tc.unexpected, output)
			}
		})
	}
}

func TestSlogLogger_ContextMethods(t *testing.T) {
	var buf bytes.Buffer
	l := NewSlogLoggerWithWriter("debug", &buf)
	ctx := context.Background()

	l.DebugContext(ctx, "debug ctx", "k1", "v1")
	l.InfoContext(ctx, "info ctx", "k2", "v2")
	l.WarnContext(ctx, "warn ctx", "k3", "v3")
	l.ErrorContext(ctx, "error ctx", "k4", "v4")

	output := buf.String()
	for _, expected := range []string{"debug ctx", "info ctx", "warn ctx", "error ctx", "k1=v1", "k2=v2", "k3=v3", "k4=v4"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestSlogLogger_With(t *testing.T) {
	var buf bytes.Buffer
	l := NewSlogLoggerWithWriter("info", &buf)
	l2 := l.With("globalKey", "globalValue")

	l2.Info("test message", "localKey", "localValue")

	output := buf.String()
	if !strings.Contains(output, "globalKey=globalValue") {
		t.Errorf("expected output to contain global context, got: %q", output)
	}
	if !strings.Contains(output, "localKey=localValue") {
		t.Errorf("expected output to contain local field, got: %q", output)
	}
}

func TestSlogLogger_Panic(t *testing.T) {
	var buf bytes.Buffer
	l := NewSlogLoggerWithWriter("info", &buf)

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic, got none")
		} else if r != "panic msg" {
			t.Errorf("expected panic value 'panic msg', got %v", r)
		}
		output := buf.String()
		if !strings.Contains(output, "panic msg") {
			t.Errorf("expected log output to contain 'panic msg', got %q", output)
		}
	}()

	l.Panic("panic msg")
}

// TestSlogLogger_Fatal runs the fatal log in a separate process to verify exit code 1.
func TestSlogLogger_Fatal(t *testing.T) {
	if os.Getenv("BE_FATAL") == "1" {
		l := NewSlogLoggerWithWriter("info", os.Stdout)
		l.Fatal("fatal msg")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestSlogLogger_Fatal")
	cmd.Env = append(os.Environ(), "BE_FATAL=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected command to exit with error")
	}

	exitError, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected exit error, got %T: %v", err, err)
	}

	if exitError.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitError.ExitCode())
	}
}
