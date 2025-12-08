package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"auto_upload_tiktok/config"
)

// Manager manages application loggers and their underlying files.
type Manager struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	infoFile    *os.File
	errorFile   *os.File
}

var global *Manager

// Initialize configures the global logger manager.
func Initialize(cfg *config.Config) (*Manager, error) {
	manager, err := New(cfg)
	if err != nil {
		return nil, err
	}
	global = manager
	return manager, nil
}

// New creates a new Manager instance.
func New(cfg *config.Config) (*Manager, error) {
	dir := cfg.LogDirectory
	if dir == "" {
		dir = "./logs"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	outputFile := cfg.LogOutputFile
	if outputFile == "" {
		outputFile = "app.log"
	}
	errorFile := cfg.LogErrorFile
	if errorFile == "" {
		errorFile = "app.error.log"
	}

	infoPath := filepath.Join(dir, outputFile)
	errPath := filepath.Join(dir, errorFile)

	infoHandle, err := os.OpenFile(infoPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open info log file: %w", err)
	}

	errorHandle, err := os.OpenFile(errPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		infoHandle.Close()
		return nil, fmt.Errorf("open error log file: %w", err)
	}

	infoWriter := io.MultiWriter(os.Stdout, infoHandle)
	errorWriter := io.MultiWriter(os.Stderr, errorHandle)

	infoLogger := log.New(infoWriter, "[INFO] ", log.LstdFlags|log.Lmicroseconds)
	errorLogger := log.New(errorWriter, "[ERROR] ", log.LstdFlags|log.Lmicroseconds)

	return &Manager{
		infoLogger:  infoLogger,
		errorLogger: errorLogger,
		infoFile:    infoHandle,
		errorFile:   errorHandle,
	}, nil
}

// Info returns the info logger.
func (m *Manager) Info() *log.Logger {
	return m.infoLogger
}

// Error returns the error logger.
func (m *Manager) Error() *log.Logger {
	return m.errorLogger
}

// Close releases file handles.
func (m *Manager) Close() error {
	var firstErr error
	if m.infoFile != nil {
		if err := m.infoFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if m.errorFile != nil {
		if err := m.errorFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Close releases the global logger manager if initialized.
func Close() error {
	if global == nil {
		return nil
	}
	err := global.Close()
	global = nil
	return err
}

// Info returns the global info logger.
func Info() *log.Logger {
	if global != nil {
		return global.Info()
	}
	return log.Default()
}

// Error returns the global error logger.
func Error() *log.Logger {
	if global != nil {
		return global.Error()
	}
	return log.Default()
}
