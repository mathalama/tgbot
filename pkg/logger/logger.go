package logger

import (
	"fmt"
	"log"
	"os"
)

// Logger простой логгер
type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
}

// New создает новый логгер
func New() *Logger {
	return &Logger{
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
	}
}

// Info логирует информационное сообщение
func (l *Logger) Info(msg string, args ...interface{}) {
	l.infoLogger.Println(fmt.Sprintf(msg, args...))
}

// Error логирует ошибку
func (l *Logger) Error(msg string, args ...interface{}) {
	l.errorLogger.Println(fmt.Sprintf(msg, args...))
}
