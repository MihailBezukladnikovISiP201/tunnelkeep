package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	appLogger *Logger
)

type Logger struct {
	mu      sync.Mutex
	logFile *os.File
	logger  *log.Logger
	path    string
}

func initLogger() (*Logger, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	dir := filepath.Join(appData, "VPNGuardian")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(dir, "guardian.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	writer := io.MultiWriter(os.Stdout, file)
	l := log.New(writer, "", 0)

	appLogger = &Logger{
		logFile: file,
		logger:  l,
		path:    logPath,
	}
	return appLogger, nil
}

func (l *Logger) Logf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	l.logger.Printf("[%s] %s\n", timestamp, msg)
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		_ = l.logFile.Close()
	}
}

func (l *Logger) GetPath() string {
	return l.path
}
