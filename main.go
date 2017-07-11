package main

import (
	"log"

	"github.com/docker/go-plugins-helpers/sdk"
)

func main() {
	sdkhandler := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	inithandlers(&sdkhandler, &FakeDriver{})

	if err := sdkhandler.ServeUnix("jsonfile", 0); err != nil {
		log.Fatal(err)
	}
}
