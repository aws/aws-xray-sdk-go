package logger

import (
	"fmt"
	"log"
)

type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
}

func init() {
	internalLogger = &LoggerImpl{}
}

var internalLogger Logger

func InjectLogger(l Logger) {
	internalLogger = l
}

type LoggerImpl struct {
	DebugLvl bool
	InfoLvl  bool
	WarnLvl  bool
}

//Debug log off by default
func (l *LoggerImpl) Debug(msg string) {
	if l.DebugLvl {
		log.Println(fmt.Sprintf("[DEBUG] %s", msg))
	}
}

func (l *LoggerImpl) Debugf(format string, args ...interface{}) {
	if l.DebugLvl {
		log.Printf(fmt.Sprintf("[DEBUG] %s", format), args)
	}
}

//Info log off by default
func (l *LoggerImpl) Info(msg string) {
	if l.DebugLvl || l.InfoLvl {
		log.Println(fmt.Sprintf("[INFO] %s", msg))
	}
}

func (l *LoggerImpl) Infof(format string, args ...interface{}) {
	if l.DebugLvl || l.InfoLvl {
		log.Printf(fmt.Sprintf("[INFO] %s", format), args)
	}
}

func (l *LoggerImpl) Warn(msg string) {
	if l.DebugLvl || l.InfoLvl || l.WarnLvl {
		log.Println(fmt.Sprintf("[WARN] %s", msg))
	}
}

func (l *LoggerImpl) Warnf(format string, args ...interface{}) {
	if l.DebugLvl || l.InfoLvl || l.WarnLvl {
		log.Printf(fmt.Sprintf("[WARN] %s", format), args)
	}
}

func (l *LoggerImpl) Error(msg string) {
	log.Println(fmt.Sprintf("[ERROR] %s", msg))
}

func (l *LoggerImpl) Errorf(format string, args ...interface{}) {
	log.Printf(fmt.Sprintf("[ERROR] %s", format), args)
}
