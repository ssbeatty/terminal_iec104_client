package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/rivo/tview"
)

type LoggerLevel string

const (
	// LoggerLevelInfo represents the info log level
	LoggerLevelInfo LoggerLevel = "info"
	// LoggerLevelDebug represents the debug log level
	LoggerLevelDebug LoggerLevel = "debug"
)

// Logger provides logging functionality for the application
type Logger struct {
	textView *tview.TextView
	mu       sync.Mutex
	Level    LoggerLevel
}

// NewLogger creates a new logger instance
func NewLogger(textView *tview.TextView, level LoggerLevel) *Logger {
	return &Logger{
		Level:    level,
		textView: textView,
	}
}

// Infof adds a log entry to the log view
func (l *Logger) Infof(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf("[white]Info: [%s] %s", timestamp, fmt.Sprintf(format, args...))

	l.textView.SetText(l.textView.GetText(false) + message)
	l.textView.ScrollToEnd()
}

// Debugf adds an error log entry to the log view
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.Level != LoggerLevelDebug {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf("[blue]Debug: [%s] %s", timestamp, fmt.Sprintf(format, args...))

	l.textView.SetText(l.textView.GetText(false) + message)
	l.textView.ScrollToEnd()
}

// Errorf adds an error log entry to the log view
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf("[red]Error: [%s] %s[red]", timestamp, fmt.Sprintf(format, args...))

	l.textView.SetText(l.textView.GetText(false) + message)
	l.textView.ScrollToEnd()
}

// Clear clears all log entries
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.textView.SetText("")
}
