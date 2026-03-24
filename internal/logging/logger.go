// Package logging provides leveled logging to stderr and an optional log file.
// It mirrors the log_message() function from the bash repo_lib.sh library.
package logging

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// Level represents log verbosity. Lower values are more severe.
type Level int

const (
	Error   Level = 0
	Warning Level = 1
	Info    Level = 2
	Debug1  Level = 3
	Debug2  Level = 4
)

const timestampFormat = "2006.01.02:15:04:05"

// Logger writes timestamped, leveled messages to stderr and an optional log file.
type Logger struct {
	verbosity Level
	stderr    io.Writer
	logFile   *os.File
}

// NewLogger creates a logger that writes to stderr and, if logPath is non-empty,
// to a log file opened with 0600 permissions. Pass an empty logPath to skip file
// logging.
func NewLogger(logPath string, verbosity Level) (*Logger, error) {
	l := &Logger{
		verbosity: verbosity,
		stderr:    os.Stderr,
	}

	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) //nolint:gosec // G304: path from trusted caller
		if err != nil {
			return nil, fmt.Errorf("open log file %s: %w", logPath, err)
		}
		l.logFile = f
	}

	return l, nil
}

// NewLoggerWithWriter creates a logger using a custom writer instead of os.Stderr.
// Intended for tests.
func NewLoggerWithWriter(w io.Writer, logPath string, verbosity Level) (*Logger, error) {
	l, err := NewLogger(logPath, verbosity)
	if err != nil {
		return nil, err
	}
	l.stderr = w
	return l, nil
}

// Log writes a formatted message if the given level is at or below the logger's
// verbosity. Output goes to both stderr and the log file (if configured).
func (l *Logger) Log(level Level, format string, args ...any) {
	if level > l.verbosity {
		return
	}

	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format(timestampFormat)
	line := fmt.Sprintf("%s - %s %s\n", ts, LevelString(level), msg)

	_, _ = fmt.Fprint(l.stderr, line)
	if l.logFile != nil {
		_, _ = fmt.Fprint(l.logFile, line)
	}
}

// Error logs at Error level.
func (l *Logger) Error(format string, args ...any) {
	l.Log(Error, format, args...)
}

// Warn logs at Warning level.
func (l *Logger) Warn(format string, args ...any) {
	l.Log(Warning, format, args...)
}

// Info logs at Info level.
func (l *Logger) Info(format string, args ...any) {
	l.Log(Info, format, args...)
}

// Debug1 logs at Debug1 level.
func (l *Logger) Debug1(format string, args ...any) {
	l.Log(Debug1, format, args...)
}

// Debug2 logs at Debug2 level.
func (l *Logger) Debug2(format string, args ...any) {
	l.Log(Debug2, format, args...)
}

// Close closes the log file, if one is open. Safe to call if no file is open.
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// LevelString returns a padded level name suitable for log output.
// All strings are padded to match the width of "WARNING:" (8 chars + colon).
func LevelString(level Level) string {
	switch level {
	case Error:
		return "ERROR:  "
	case Warning:
		return "WARNING:"
	case Info:
		return "INFO:   "
	case Debug1:
		return "DEBUG1: "
	case Debug2:
		return "DEBUG2: "
	default:
		return "UNKNOWN:"
	}
}

// VerbosityFromEnv reads the OW_VERBOSITY environment variable, parses it as
// an integer, and clamps it to the range [0, 4]. Returns Info (2) if the
// variable is missing or unparseable.
func VerbosityFromEnv() Level {
	val := os.Getenv("OW_VERBOSITY")
	if val == "" {
		return Info
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return Info
	}

	if n < int(Error) {
		return Error
	}
	if n > int(Debug2) {
		return Debug2
	}
	return Level(n)
}
