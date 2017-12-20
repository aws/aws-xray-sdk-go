package logger

func Debug(msg string) {
	internalLogger.Debug(msg)
}

func Debugf(format string, args ...interface{}) {
	internalLogger.Debugf(format, args)
}

func Info(msg string) {
	internalLogger.Info(msg)
}

func Infof(format string, args ...interface{}) {
	internalLogger.Infof(format, args)
}

func Warn(msg string) {
	internalLogger.Warn(msg)
}

func Warnf(format string, args ...interface{}) {
	internalLogger.Warnf(format, args)
}

func Error(msg string) {
	internalLogger.Error(msg)
}

func Errorf(format string, args ...interface{}) {
	internalLogger.Errorf(format, args)
}
