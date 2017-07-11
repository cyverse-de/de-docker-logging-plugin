package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	protoio "github.com/gogo/protobuf/io"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fifo"
)

// LoggingDriver defines the interface for types that want to be a Docker logging
// plugin for the DE.
type LoggingDriver interface {
	StartLogging(file string, info logger.Info) error
	StopLogging(file string) error
}

// FakeDriver doesn't actually do anything except log when it receives a message.
type FakeDriver struct{}

// StartLogging doesn't do anything except log that it was called.
func (f *FakeDriver) StartLogging(file string, info logger.Info) error {
	log.Printf("StartLogging called with file %s for container %s\n", file, info.ContainerID)
	return nil
}

// StopLogging doesn't do anything except log that it was called.
func (f *FakeDriver) StopLogging(file string) error {
	log.Printf("StopLogging was called with file %s\n", file)
	return nil
}

// FileLogger tracks the info needed to write out log messages to files.
type FileLogger struct {
	StderrPath string
	StdoutPath string
	Stdout     *os.File
	Stderr     *os.File
	LogStream  io.ReadCloser
}

func (l *FileLogger) StreamMessages() {
	reader := protoio.NewUint32DelimitedReader(l.LogStream, binary.BigEndian, 1e6)
	defer reader.Close()

	var (
		err   error
		entry logdriver.LogEntry
	)

	for {
		if err = reader.ReadMsg(&entry); err != nil {
			if err == io.EOF {
				l.LogStream.Close()
				return
			}
			reader = protoio.NewUint32DelimitedReader(l.LogStream, binary.BigEndian, 1e6)
		}

		msg := logger.Message{
			Line:      entry.Line,
			Source:    entry.Source,
			Partial:   entry.Partial,
			Timestamp: time.Unix(0, entry.TimeNano),
		}

		switch msg.Source {
		case "stderr":
			if _, err = l.Stderr.Write(msg.Line); err != nil {
				err = errors.Wrap(err, "error writing to stderr log file")
				log.Println(err.Error())
				continue
			}
		case "stdout":
			if _, err = l.Stdout.Write(msg.Line); err != nil {
				err = errors.Wrap(err, "error writing to stdout log file")
				log.Println(err.Error())
				continue
			}
		default:
			log.Println(fmt.Errorf("Unknown source %s for message: %s", msg.Source, msg.Line))
			continue
		}

		entry.Reset()
	}
}

// FileDriver is a logging driver that will write out the stderr and stdout
// streams to a configured directory.
type FileDriver struct {
	mu     sync.Mutex
	logmap map[string]*FileLogger
}

// NewFileDriver returns a newly created *FileDriver.
func NewFileDriver() *FileDriver {
	return &FileDriver{
		logmap: make(map[string]*FileLogger),
	}
}

// StartLogging sets up everything needed for logging to separate files for
// stdout and stderr. Fires up a goroutine that pipes info from the FIFO created
// by Docker into each file.
func (d *FileDriver) StartLogging(fifopath string, loginfo logger.Info) error {
	d.mu.Lock()
	if _, ok := d.logmap[fifopath]; ok {
		d.mu.Unlock()
		return fmt.Errorf("logging is already configured for %s", fifopath)
	}
	d.mu.Unlock()

	if _, ok := loginfo.Config["stderr"]; !ok {
		return fmt.Errorf("'stderr' path missing from the plugin configuration")
	}

	if _, ok := loginfo.Config["stdout"]; !ok {
		return fmt.Errorf("'stdout' path missing from the plugin configuration")
	}

	stderrPath := loginfo.Config["stderr"]
	stdoutPath := loginfo.Config["stdout"]

	stderrBase := path.Base(stderrPath)
	stdoutBase := path.Base(stdoutPath)

	for _, p := range []string{stderrBase, stdoutBase} {
		pinfo, err := os.Stat(p)
		if err != nil {
			return errors.Wrapf(err, "error stating path %s", p)
		}
		if !pinfo.IsDir() {
			return errors.Wrapf(err, "path was not a directory %s", p)
		}
	}

	stderr, err := os.Create(stderrPath)
	if err != nil {
		return errors.Wrapf(err, "error opening stderr log file at %s", stderrBase)
	}

	stdout, err := os.Create(stdoutPath)
	if err != nil {
		return errors.Wrapf(err, "error opening stdout log file at %s", stdoutBase)
	}

	f, err := fifo.OpenFifo(context.Background(), fifopath, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening fifo file %s", fifopath)
	}

	filelogger := &FileLogger{
		StderrPath: stderrPath,
		StdoutPath: stdoutPath,
		Stderr:     stderr,
		Stdout:     stdout,
		LogStream:  f,
	}

	d.mu.Lock()
	d.logmap[fifopath] = filelogger
	d.mu.Unlock()

	go filelogger.StreamMessages()

	return nil
}

// StopLogging terminates logging to files and closes them out.
func (d *FileDriver) StopLogging(fifopath string) error {
	d.mu.Lock()
	fl, ok := d.logmap[fifopath]
	if ok {
		fl.LogStream.Close()
		fl.Stderr.Close()
		fl.Stdout.Close()
		delete(d.logmap, fifopath)
	}
	d.mu.Unlock()
	return nil
}