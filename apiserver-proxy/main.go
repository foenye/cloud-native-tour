package main

import (
	"github.com/yeahfo/cloud-native-tour/apiserver-proxy/pkg/cmd"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
	"os"
)

func main() {
	stopSignal := server.SetupSignalHandler()
	command := cmd.NewProxyServerCommand(stopSignal)
	code := cli.Run(command)
	os.Exit(code)
}
