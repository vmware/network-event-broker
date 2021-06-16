/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package log

import (
	"io/ioutil"
	"log"
	"os"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)

var (
	logDebug *log.Logger
	logInfo  *log.Logger
	logWarn  *log.Logger
	logError *log.Logger
	logFatal *log.Logger
)

var Level string

func initDefault() {
	logInfo = log.New(os.Stdout, "[ "+LogLevelInfo+" ] ", log.LstdFlags)
	logWarn = log.New(os.Stdout, "[ "+LogLevelWarn+" ] ", log.LstdFlags)
	logError = log.New(os.Stderr, "[ "+LogLevelError+" ] ", log.LstdFlags)
	logFatal = log.New(os.Stdout, "[ "+LogLevelFatal+" ] ", log.LstdFlags)
}

func Init() {
	SetLevel("info")
}

func SetLevel(level string) {
	switch level {
	case LogLevelDebug:

		logDebug = log.New(os.Stdout, "[ "+LogLevelDebug+" ] ", log.LstdFlags)
		logInfo = log.New(os.Stdout, "[ "+LogLevelInfo+" ] ", log.LstdFlags)
		logWarn = log.New(os.Stdout, "[ "+LogLevelWarn+" ] ", log.LstdFlags)
		logError = log.New(os.Stderr, "[ "+LogLevelError+" ] ", log.LstdFlags)
		logFatal = log.New(os.Stdout, "["+LogLevelFatal+" ] ", log.LstdFlags)

	case LogLevelInfo:
		initDefault()

	case LogLevelWarn:
		logWarn = log.New(os.Stdout, "[ "+LogLevelWarn+" ] ", log.LstdFlags)
		logError = log.New(os.Stderr, "[ "+LogLevelError+" ] ", log.LstdFlags)
		logFatal = log.New(os.Stdout, "[ "+LogLevelFatal+" ] ", log.LstdFlags)

	case LogLevelError:
		logError = log.New(os.Stderr, "[ "+LogLevelError+" ] ", log.LstdFlags)
		logFatal = log.New(os.Stdout, "[ "+LogLevelFatal+" ] ", log.LstdFlags)

	case LogLevelFatal:
		logError = log.New(ioutil.Discard, "[ "+LogLevelError+" ] ", log.LstdFlags)
		logFatal = log.New(os.Stdout, "[ "+LogLevelFatal+" ] ", log.LstdFlags)

	default:
		initDefault()
	}
}

func Debugf(format string, v ...interface{}) {
	if logDebug != nil {
		logDebug.Printf(format, v...)
	}
}

func Debugln(v ...interface{}) {
	if logDebug != nil {
		logDebug.Println(v...)
	}
}

func Infof(format string, v ...interface{}) {
	if logInfo != nil {
		logInfo.Printf(format, v...)
	}
}

func Infoln(v ...interface{}) {
	if logInfo != nil {
		logInfo.Println(v...)
	}
}

func Warnf(format string, v ...interface{}) {
	if logWarn != nil {
		logError.Printf(format, v...)
	}
}

func Warnln(v ...interface{}) {
	if logWarn != nil {
		logError.Println(v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if logError != nil {
		logError.Printf(format, v...)
	}
}

func Errorln(v ...interface{}) {
	if logError != nil {
		logError.Println(v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	if logFatal != nil {
		logInfo.Printf(format, v...)

	}
}

func Fatalln(v ...interface{}) {
	if logFatal != nil {
		logFatal.Println(v...)
	}
}
