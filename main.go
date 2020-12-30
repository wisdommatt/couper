//go:generate go run assets/generate/generate.go

package main

import (
	"flag"
	"os"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
)

func main() {
	fields := logrus.Fields{
		"type":    "couper_daemon",
		"build":   runtime.BuildName,
		"version": runtime.VersionName,
	}

	args := command.NewArgs()
	if len(args) == 0 || command.NewCommand(args[0]) == nil {
		command.Help()
		os.Exit(1)
	}

	var filename, logFormat string
	set := flag.NewFlagSet("global", flag.ContinueOnError)
	set.StringVar(&filename, "f", config.DefaultFileName, "-f ./couper.hcl")
	set.StringVar(&logFormat, "log-format", config.DefaultSettings.LogFormat, "-log-format=common")
	err := set.Parse(args.Filter(set))
	if err != nil {
		logrus.WithFields(fields).Fatal(err)
	}

	logger := newLogger(logFormat).WithFields(fields)

	wd, err := config.SetWorkingDirectory(filename)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Infof("working directory: %s", wd)

	confFile, err := configload.LoadFile(path.Base(filename))
	if err != nil {
		logger.Fatal(err)
	}

	var exitCode int
	if err = command.NewCommand(args[0]).Execute(args, confFile, logger); err != nil {
		logger.Error(err)
		exitCode = 1
	}
	logrus.Exit(exitCode)
}

func newLogger(format string) logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = os.Stdout

	if format == "json" {
		logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "timestamp",
			logrus.FieldKeyMsg:  "message",
		}}
	}
	logger.Level = logrus.DebugLevel
	return logger
}
