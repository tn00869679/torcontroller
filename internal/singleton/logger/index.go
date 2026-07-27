package logger

import (
	"log"
	"os"
	"sync"
)

var (
	instance    *Logger
	once        sync.Once
	logFilePath = "/var/log/torcontroller.log"
)

type Logger struct {
	Logger *log.Logger
}

func GetLogger() *Logger {
	once.Do(func() {
		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}

		baseLogger := log.New(logFile, "TORCONTROLLER: ", log.Ldate|log.Ltime|log.Lshortfile)
		instance = &Logger{Logger: baseLogger}
	})
	return instance
}

// Info logs for recording information levels
func (l *Logger) Info(message string) {
	l.Logger.Printf("[INFO] %s", message)
}

// Warn logs for recording information levels
func (l *Logger) Warn(message string) {
	l.Logger.Printf("[WARN] %s", message)
}

// Error log of error levels
func (l *Logger) Error(message string) {
	l.Logger.Printf("[ERROR] %s", message)
}
