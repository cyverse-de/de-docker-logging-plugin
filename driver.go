package main

import (
	"log"

	"github.com/docker/docker/daemon/logger"
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
