package logger

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const (
	// FATAL represents fatal error
	FATAL = iota
	// CRITICAL represents critical error
	CRITICAL
	// WARNING represents warning which should be taken a look
	WARNING
	// INFO represents normal information
	INFO
	// DEBUG represents information used to debug
	DEBUG
	// DISABLE represents there is nothing filled to log file
	DISABLE
)

// LOGLEVEL is a map of level value and corresponding string
var LOGLEVEL = map[int]string{
	FATAL:    "FATAL",
	CRITICAL: "CRITICAL",
	WARNING:  "WARNING",
	INFO:     "INFO",
	DEBUG:    "DEBUG",
	DISABLE:  "DISABLE",
}

// Logging contains 5 loggers with configureable log level, prefix and stream
type Logging struct {
	Fatal    *log.Logger
	Critical *log.Logger
	Warning  *log.Logger
	Info     *log.Logger
	Debug    *log.Logger
	level    int
	stream   io.Writer
	prefix   string
}

var instance *Logging
var once sync.Once

// New initializes singleton logger
func New() *Logging {
	once.Do(func() {
		instance = &Logging{}
		instance.level = INFO
		instance.prefix = ""
		instance.stream = os.Stderr

		instance.Fatal = log.New(
			instance.stream,
			"FATAL   : ",
			log.Ldate|log.Lmicroseconds)
		instance.Critical = log.New(
			instance.stream,
			"CRITICAL: ",
			log.Ldate|log.Lmicroseconds|log.Lshortfile)
		instance.Warning = log.New(
			instance.stream,
			"WARNING : ",
			log.Ldate|log.Lmicroseconds)
		instance.Info = log.New(
			instance.stream,
			"INFO    : ",
			log.Ldate|log.Lmicroseconds)
		instance.Debug = log.New(
			instance.stream,
			"DEBUG   : ",
			log.Ldate|log.Lmicroseconds|log.Lshortfile)

		instance.SetStreamSingle(os.Stderr)
	})

	return instance
}

// SetLevel configures minimal log level will be displayed
func (l *Logging) SetLevel(level int) {
	l.level = level
	switch level {
	case FATAL:
		l.Fatal.SetOutput(l.stream)
		l.Critical.SetOutput(ioutil.Discard)
		l.Warning.SetOutput(ioutil.Discard)
		l.Info.SetOutput(ioutil.Discard)
		l.Debug.SetOutput(ioutil.Discard)

	case CRITICAL:
		l.Fatal.SetOutput(l.stream)
		l.Critical.SetOutput(l.stream)
		l.Warning.SetOutput(ioutil.Discard)
		l.Info.SetOutput(ioutil.Discard)
		l.Debug.SetOutput(ioutil.Discard)

	case WARNING:
		l.Fatal.SetOutput(l.stream)
		l.Critical.SetOutput(l.stream)
		l.Warning.SetOutput(l.stream)
		l.Info.SetOutput(ioutil.Discard)
		l.Debug.SetOutput(ioutil.Discard)

	case INFO:
		l.Fatal.SetOutput(l.stream)
		l.Critical.SetOutput(l.stream)
		l.Warning.SetOutput(l.stream)
		l.Info.SetOutput(l.stream)
		l.Debug.SetOutput(ioutil.Discard)

	case DEBUG:
		l.Fatal.SetOutput(l.stream)
		l.Critical.SetOutput(l.stream)
		l.Warning.SetOutput(l.stream)
		l.Info.SetOutput(l.stream)
		l.Debug.SetOutput(l.stream)

	case DISABLE:
		l.Fatal.SetOutput(ioutil.Discard)
		l.Critical.SetOutput(ioutil.Discard)
		l.Warning.SetOutput(ioutil.Discard)
		l.Info.SetOutput(ioutil.Discard)
		l.Debug.SetOutput(ioutil.Discard)
	}
}

// SetPrefix configures prefix of each line of log
func (l *Logging) SetPrefix(pfix string) {
	l.prefix = pfix
	l.Fatal.SetPrefix(pfix + " " + l.Fatal.Prefix())
	l.Critical.SetPrefix(pfix + " " + l.Critical.Prefix())
	l.Warning.SetPrefix(pfix + " " + l.Warning.Prefix())
	l.Info.SetPrefix(pfix + " " + l.Info.Prefix())
	l.Debug.SetPrefix(pfix + " " + l.Debug.Prefix())
}

// SetStreamSingle configure to log to only one stream
func (l *Logging) SetStreamSingle(stream io.Writer) {
	l.stream = stream
	l.SetLevel(l.level)
}

// SetStreamMulti configures to log to multiple streams
func (l *Logging) SetStreamMulti(streams []io.Writer) {
	l.stream = io.MultiWriter(streams...)
	l.SetLevel(l.level)
}
