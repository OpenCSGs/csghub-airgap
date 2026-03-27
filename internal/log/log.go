package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	Log *logrus.Logger
	Std *log.Logger
)

type simpleFormatter struct{}

func (f *simpleFormatter) Format(e *logrus.Entry) ([]byte, error) {
	// Force UTC+8 timezone and show milliseconds (3 digits)
	loc := time.FixedZone("UTC+8", 8*3600)
	ts := e.Time.In(loc).Format("2006-01-02T15:04:05.000+08:00")
	level := strings.ToUpper(e.Level.String())
	msg := e.Message
	return []byte(fmt.Sprintf("%s[%s] %s\n", level, ts, msg)), nil
}

func init() {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)
	Log.SetFormatter(&simpleFormatter{})
}

func Init(debug bool) {
	if debug {
		Log.SetLevel(logrus.DebugLevel)
		Std = log.New(Log.Out, "[HELM]", log.LstdFlags)
	} else {
		Log.SetLevel(logrus.InfoLevel)
		Std = log.New(io.Discard, "[HELM]", log.LstdFlags)
	}
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

func Fatal(args ...interface{}) {
	Log.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	Log.Fatalf(format, args...)
}

func SetOutput(w io.Writer) {
	if w == nil {
		return
	}

	Log.SetOutput(w)
	// Re-apply formatter to avoid fallback to default logrus format
	Log.SetFormatter(&simpleFormatter{})
	if Std != nil {
		Std.SetOutput(w)
	}
}
