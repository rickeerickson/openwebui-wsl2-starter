package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// timestampPattern matches YYYY.MM.DD:HH:MM:SS.
var timestampPattern = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}:\d{2}:\d{2}:\d{2}`)

func TestLevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		verbosity Level
		logLevel  Level
		wantLog   bool
	}{
		{"error at error verbosity", Error, Error, true},
		{"warning at error verbosity", Error, Warning, false},
		{"info at info verbosity", Info, Info, true},
		{"debug1 at info verbosity", Info, Debug1, false},
		{"debug2 at debug2 verbosity", Debug2, Debug2, true},
		{"error at debug2 verbosity", Debug2, Error, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l, err := NewLoggerWithWriter(&buf, "", tt.verbosity)
			if err != nil {
				t.Fatalf("NewLoggerWithWriter: %v", err)
			}

			l.Log(tt.logLevel, "test message")

			got := buf.String()
			if tt.wantLog && got == "" {
				t.Errorf("expected output but got none")
			}
			if !tt.wantLog && got != "" {
				t.Errorf("expected no output but got: %s", got)
			}
		})
	}
}

func TestTimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLoggerWithWriter(&buf, "", Info)
	if err != nil {
		t.Fatalf("NewLoggerWithWriter: %v", err)
	}

	l.Info("timestamp check")

	line := buf.String()
	if !timestampPattern.MatchString(line) {
		t.Errorf("output does not match timestamp pattern YYYY.MM.DD:HH:MM:SS:\n%s", line)
	}
}

func TestLevelPrefixAlignment(t *testing.T) {
	tests := []struct {
		level  Level
		prefix string
	}{
		{Error, "ERROR:  "},
		{Warning, "WARNING:"},
		{Info, "INFO:   "},
		{Debug1, "DEBUG1: "},
		{Debug2, "DEBUG2: "},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			got := LevelString(tt.level)
			if got != tt.prefix {
				t.Errorf("LevelString(%d) = %q, want %q", tt.level, got, tt.prefix)
			}
			if len(got) != 8 {
				t.Errorf("LevelString(%d) length = %d, want 8", tt.level, len(got))
			}
		})
	}
}

func TestLevelStringUnknown(t *testing.T) {
	got := LevelString(Level(99))
	if got != "UNKNOWN:" {
		t.Errorf("LevelString(99) = %q, want %q", got, "UNKNOWN:")
	}
}

func TestLogFilePermissions(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	l, err := NewLogger(logPath, Info)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	l.Info("permissions check")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("log file permissions = %o, want 0600", perm)
	}
}

func TestVerbosityFromEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  Level
	}{
		{"valid 0", "0", true, Error},
		{"valid 4", "4", true, Debug2},
		{"valid 2", "2", true, Info},
		{"invalid string", "abc", true, Info},
		{"missing var", "", false, Info},
		{"below range", "-1", true, Error},
		{"above range", "10", true, Debug2},
		{"valid 1", "1", true, Warning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("OW_VERBOSITY", tt.value)
			} else {
				_ = os.Unsetenv("OW_VERBOSITY")
			}

			got := VerbosityFromEnv()
			if got != tt.want {
				t.Errorf("VerbosityFromEnv() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewLoggerEmptyPathSkipsFile(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLoggerWithWriter(&buf, "", Info)
	if err != nil {
		t.Fatalf("NewLoggerWithWriter: %v", err)
	}

	if l.logFile != nil {
		t.Error("expected logFile to be nil when path is empty")
	}

	l.Info("no file")
	if buf.String() == "" {
		t.Error("expected stderr output even without log file")
	}

	if err := l.Close(); err != nil {
		t.Errorf("Close on nil logFile returned error: %v", err)
	}
}

func TestCloseFlushesFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "close.log")

	l, err := NewLogger(logPath, Info)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	l.Info("before close")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.Contains(string(data), "before close") {
		t.Error("log file missing expected content after Close")
	}
}

func TestDualOutput(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "dual.log")

	var buf bytes.Buffer
	l, err := NewLoggerWithWriter(&buf, logPath, Info)
	if err != nil {
		t.Fatalf("NewLoggerWithWriter: %v", err)
	}

	l.Info("dual output test")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	stderrOut := buf.String()
	if !strings.Contains(stderrOut, "dual output test") {
		t.Error("stderr missing expected message")
	}

	fileData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.Contains(string(fileData), "dual output test") {
		t.Error("log file missing expected message")
	}

	if stderrOut != string(fileData) {
		t.Errorf("stderr and file output differ:\nstderr: %s\nfile:   %s", stderrOut, string(fileData))
	}
}

func TestInjectableWriter(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLoggerWithWriter(&buf, "", Debug2)
	if err != nil {
		t.Fatalf("NewLoggerWithWriter: %v", err)
	}

	l.Error("e %d", 1)
	l.Warn("w %d", 2)
	l.Info("i %d", 3)
	l.Debug1("d1 %d", 4)
	l.Debug2("d2 %d", 5)

	output := buf.String()
	for _, want := range []string{"e 1", "w 2", "i 3", "d1 4", "d2 5"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestConvenienceMethods(t *testing.T) {
	tests := []struct {
		name   string
		call   func(l *Logger)
		prefix string
	}{
		{"Error", func(l *Logger) { l.Error("msg") }, "ERROR:"},
		{"Warn", func(l *Logger) { l.Warn("msg") }, "WARNING:"},
		{"Info", func(l *Logger) { l.Info("msg") }, "INFO:"},
		{"Debug1", func(l *Logger) { l.Debug1("msg") }, "DEBUG1:"},
		{"Debug2", func(l *Logger) { l.Debug2("msg") }, "DEBUG2:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l, err := NewLoggerWithWriter(&buf, "", Debug2)
			if err != nil {
				t.Fatalf("NewLoggerWithWriter: %v", err)
			}

			tt.call(l)

			if !strings.Contains(buf.String(), tt.prefix) {
				t.Errorf("output missing prefix %q: %s", tt.prefix, buf.String())
			}
		})
	}
}

func TestNewLoggerInvalidPath(t *testing.T) {
	_, err := NewLogger("/nonexistent/dir/test.log", Info)
	if err == nil {
		t.Error("expected error for invalid log path, got nil")
	}
}

func TestOutputFormat(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLoggerWithWriter(&buf, "", Info)
	if err != nil {
		t.Fatalf("NewLoggerWithWriter: %v", err)
	}

	l.Error("something broke")

	line := buf.String()
	// Expected format: "YYYY.MM.DD:HH:MM:SS - ERROR:   something broke\n"
	pattern := regexp.MustCompile(
		`^\d{4}\.\d{2}\.\d{2}:\d{2}:\d{2}:\d{2} - ERROR:   something broke\n$`,
	)
	if !pattern.MatchString(line) {
		t.Errorf("output format mismatch:\ngot:  %q\nwant: YYYY.MM.DD:HH:MM:SS - ERROR:   something broke\\n", line)
	}
}
