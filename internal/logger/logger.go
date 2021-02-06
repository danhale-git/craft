package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
)

// Global loggers
//nolint:gochecknoglobals
var (
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

// Init creates loggers
func Init(path, level, prefix string) {
	flags := log.LstdFlags

	if prefix != "" && !strings.HasPrefix(prefix, " ") {
		prefix = " " + prefix
	}

	Info = log.New(ioutil.Discard, fmt.Sprintf("[Info]%s ", prefix), flags)
	Warn = log.New(ioutil.Discard, fmt.Sprintf("[Warn]%s", prefix), flags)
	Error = log.New(ioutil.Discard, fmt.Sprintf("[Error]%s ", prefix), flags)

	var out io.Writer

	if path != "" {
		var err error
		out, err = rotatelogs.New(path)

		if err != nil {
			log.Fatalf("Error creating log rotator: %s", err)
		}
	} else {
		out = os.Stdout
	}

	switch strings.ToLower(level) {
	case "info":
		Info.SetOutput(out)
	case "warn":
		Warn.SetOutput(out)
	case "error":
		Error.SetOutput(out)
	default:
		log.Fatalf("Invalid error level '%s': expected info|warn|error", level)
	}
}
