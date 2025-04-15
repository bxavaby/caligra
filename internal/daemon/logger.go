// BYZRA ⸻ internal/daemon/logger.go
// daemon logging functionality

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// severity of log entries
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
)

// daemon activity logging
type Logger struct {
	logFile     *os.File
	level       LogLevel
	initialized bool
	path        string
}

func NewLogger(logPath string, level LogLevel) (*Logger, error) {
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		logFile:     logFile,
		level:       level,
		initialized: true,
		path:        logPath,
	}, nil
}

// writes a message to the log with timestamp
func (l *Logger) Log(level LogLevel, message string) error {
	if !l.initialized {
		return fmt.Errorf("logger not initialized")
	}

	if level < l.level {
		return nil // skip those below threshold
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := getLevelString(level)
	logLine := fmt.Sprintf("[%s] %s: %s\n", timestamp, levelStr, message)

	_, err := l.logFile.WriteString(logLine)
	return err
}

// debug logs
func (l *Logger) Debug(message string) error {
	return l.Log(LevelDebug, message)
}

// info logs
func (l *Logger) Info(message string) error {
	return l.Log(LevelInfo, message)
}

// warning logs
func (l *Logger) Warning(message string) error {
	return l.Log(LevelWarning, message)
}

// error logs
func (l *Logger) Error(message string) error {
	return l.Log(LevelError, message)
}

// close properly
func (l *Logger) Close() error {
	if !l.initialized || l.logFile == nil {
		return nil
	}

	err := l.logFile.Close()
	l.initialized = false
	l.logFile = nil
	return err
}

// new log file and archives the old one
func (l *Logger) Rotate() error {
	if !l.initialized {
		return fmt.Errorf("logger not initialized")
	}

	if err := l.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	newPath := fmt.Sprintf("%s.%s", l.path, timestamp)
	if err := os.Rename(l.path, newPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	logFile, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	l.logFile = logFile
	l.initialized = true

	// log rotation
	return l.Info(fmt.Sprintf("Log rotated, previous log saved as %s", newPath))
}

// converts log level 2 string
func getLevelString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
