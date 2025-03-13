package main

import (
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/cmd/server"
	genericAPIServer "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
	"os"

	// force compilation of packages we'll later rely upon
	_ "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/install"
	_ "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/validation"
	_ "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset"
	_ "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	_ "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1beta1"
)

func main() {
	context := genericAPIServer.SetupSignalContext()
	defaultOptions := server.NewDefaultOptions(os.Stdout, os.Stderr)
	command := server.NewCommandStartAggregator(context, defaultOptions)
	code := cli.Run(command)
	os.Exit(code)
}
