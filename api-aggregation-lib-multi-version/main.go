package main

import (
	"github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/cmd"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
	"os"
)

func main() {
	stopCh := server.SetupSignalHandler()
	command := cmd.NewHelloServerCommand(stopCh)
	code := cli.Run(command)
	os.Exit(code)
}
