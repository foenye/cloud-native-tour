package main

import (
	"github.com/yeahfo/cloud-native-tour/api-aggregation-lib-v1/pkg/cmd"
	genericAPIServer "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
	"os"
)

func main() {
	stopCh := genericAPIServer.SetupSignalHandler()
	command := cmd.NewHelloServerCommand(stopCh)
	code := cli.Run(command)
	os.Exit(code)
}
