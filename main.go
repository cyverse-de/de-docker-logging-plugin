package main

import (
	"log"

	"github.com/docker/go-plugins-helpers/sdk"
)

func main() {
	var err error
	pluginName := "delogplugin"
	sdkhandler := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	inithandlers(&sdkhandler, NewFileDriver())

	if err = sdkhandler.ServeUnix(pluginName, 0); err != nil {
		log.Fatal(err)
	}

}
