package log

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	Log *logrus.Logger
	Std *log.Logger
)

func init() {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	Std = log.New(Log.Out, "", log.LstdFlags)
}

// Info logs a message at level Info
func Info(args ...interface{}) {
	Log.Info(args...)
}

// Infof logs a formatted message at level Info
func Infof(format string, args ...interface{}) {
	Log.Infof(format, args...)
}

func Debug(args ...interface{}) {
	Log.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	Log.Debugf(format, args...)
}

func Warn(args ...interface{}) {
	Log.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	Log.Warnf(format, args...)
}

func Error(args ...interface{}) {
	Log.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	Log.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	Log.Fatalf(format, args...)
}
