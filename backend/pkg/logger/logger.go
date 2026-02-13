package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func Init(level, format, output string) error {
	Log = logrus.New()
	
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	Log.SetLevel(lvl)
	
	if format == "json" {
		Log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}
	
	switch output {
	case "stdout":
		Log.SetOutput(os.Stdout)
	case "stderr":
		Log.SetOutput(os.Stderr)
	default:
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		Log.SetOutput(file)
	}
	
	return nil
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}

func Info(args ...interface{}) {
	Log.Info(args...)
}

func Error(args ...interface{}) {
	Log.Error(args...)
}

func Warn(args ...interface{}) {
	Log.Warn(args...)
}

func Debug(args ...interface{}) {
	Log.Debug(args...)
}

func Fatal(args ...interface{}) {
	Log.Fatal(args...)
}
