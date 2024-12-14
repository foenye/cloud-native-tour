package cmd

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	myAPIserver "github.com/yeahfo/cloud-native-tour/api-aggregation-lib-v1/pkg/apiserver"
	generatedOpenAPI "github.com/yeahfo/cloud-native-tour/api/generated/openapi"
	hellov1 "github.com/yeahfo/cloud-native-tour/api/hello.yeahfo.github.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	waitUtil "k8s.io/apimachinery/pkg/util/wait"
	openapiNamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericAPIServer "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	cliFlag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	"net"
	"os"
)

type Options struct {
	SecureServing *options.SecureServingOptionsWithLoopback
	Kubeconfig    string
	Features      *options.FeatureOptions

	EnableEtcdStorage bool
	Etcd              *options.EtcdOptions
	options.EtcdOptions
}

func (opts *Options) Flags() (flagSets cliFlag.NamedFlagSets) {
	flagSet := flagSets.FlagSet(hellov1.Group + "-server")
	flagSet.StringVar(&opts.Kubeconfig, "kubeconfig", opts.Kubeconfig, "The path to the kubeconfig used "+
		"to connect to the Kubernetes API server and the Kubelets (defaults to in-cluster config)")
	opts.SecureServing.AddFlags(flagSets.FlagSet("apiserver secure serving"))
	opts.Features.AddFlags(flagSets.FlagSet("features"))

	flagSet.BoolVar(&opts.EnableEtcdStorage, "enable-etcd-storage", false, "If true, store objects "+
		"in etcd.")
	opts.Etcd.AddFlags(flagSets.FlagSet("Etcd"))
	return flagSets
}

// Complete fills in fields required to have valid data.
func (opts *Options) Complete() error {
	return nil
}

// Validate the ServerOptions
func (opts *Options) Validate(_ []string) error {
	return nil
}

type ServerConfig struct {
	APIServer *genericAPIServer.Config
	REST      *rest.Config
}

func (opts *Options) ServerConfig() (*myAPIserver.Config, error) {
	apiServerConfig, err := opts.APIServerConfig()
	if err != nil {
		return nil, err
	}

	if opts.EnableEtcdStorage {
		if err := opts.Etcd.ApplyWithStorageFactoryTo(storage.NewDefaultStorageFactory(
			opts.Etcd.StorageConfig,
			opts.Etcd.DefaultStorageMediaType,
			myAPIserver.Codecs,
			storage.NewDefaultResourceEncodingConfig(myAPIserver.Scheme),
			apiServerConfig.MergedResourceConfig,
			nil), &apiServerConfig.Config); err != nil {
			return nil, err
		}
		klog.Infof("etcd config: %v", opts.Etcd)
	}

	return &myAPIserver.Config{
		GenericConfig: apiServerConfig,
		ExtraConfig: myAPIserver.ExtraConfig{
			EnableEtcdStorage: opts.EnableEtcdStorage,
		},
	}, nil
}

func (opts *Options) APIServerConfig() (*genericAPIServer.RecommendedConfig, error) {
	if err := opts.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil,
		[]net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}
	serverConfig := genericAPIServer.NewRecommendedConfig(myAPIserver.Codecs)
	if err := opts.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	// Enable OpenAPI schemas
	openapiDefinitionNamer := openapiNamer.NewDefinitionNamer(myAPIserver.Scheme)
	//serverConfig.OpenAPIConfig = genericAPIServer.DefaultOpenAPIConfig(generatedOpenAPI.GetOpenAPIDefinitions, openapiDefinitionNamer)
	//serverConfig.OpenAPIConfig.Info.Title = hellov1.Group + "-server"
	//serverConfig.OpenAPIConfig.Info.Version = "1.0"
	//if feature.DefaultFeatureGate.Enabled("OpenAPIV3") {
	serverConfig.OpenAPIV3Config = genericAPIServer.DefaultOpenAPIV3Config(generatedOpenAPI.GetOpenAPIDefinitions, openapiDefinitionNamer)
	serverConfig.OpenAPIV3Config.Info.Title = hellov1.Group + "-server"
	serverConfig.OpenAPIV3Config.Info.Version = "1.0"
	//}

	return serverConfig, nil
}

func (opts *Options) restConfig() (restConfig *rest.Config, err error) {
	if len(opts.Kubeconfig) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: opts.Kubeconfig}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		restConfig, err = loader.ClientConfig()
	} else {
		restConfig, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("unable to construct lister client config: %v", err)
	}

	// Use proto buffers for communication with apiserver
	restConfig.ContentType = "application/vnd.kubernetes.protobuf"
	_ = rest.SetKubernetesDefaults(restConfig)
	return restConfig, nil
}

// NewHelloServerCommand provides a CLI handler for the metrics server entrypoint.
func NewHelloServerCommand(stopCh <-chan struct{}) *cobra.Command {
	opts := &Options{
		SecureServing: options.NewSecureServingOptions().WithLoopback(),
		Etcd:          options.NewEtcdOptions(storagebackend.NewDefaultConfig("/registry/"+hellov1.Group, nil)),
	}

	opts.Etcd.StorageConfig.EncodeVersioner = runtime.NewMultiGroupVersioner(hellov1.SchemeGroupVersion,
		schema.GroupKind{Group: hellov1.Group})
	opts.Etcd.DefaultStorageMediaType = "application/json"
	opts.SecureServing.BindPort = 6443

	command := &cobra.Command{
		Short: "Launch " + hellov1.Group + "-server",
		Long:  "Launch " + hellov1.Group + "-server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(); err != nil {
				return err
			}
			if err := opts.Validate(args); err != nil {
				return err
			}
			if err := runCommand(opts, stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := command.Flags()
	flagSets := opts.Flags()
	for _, flagSet := range flagSets.FlagSets {
		flags.AddFlagSet(flagSet)
	}
	local := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(local)
	flagSets.FlagSet("logging").AddGoFlagSet(local)

	usageFmt := "Usage:\n	%s\n"
	cols, _, _ := term.TerminalSize(command.OutOrStdout())
	command.SetHelpFunc(func(command *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(command.OutOrStdout(), usageFmt, command.UseLine())
		cliFlag.PrintSections(command.OutOrStdout(), flagSets, cols)
	})
	return command
}

func runCommand(opts *Options, stopCh <-chan struct{}) error {
	serverConfig, err := opts.ServerConfig()
	if err != nil {
		return err
	}
	apiServer, err := serverConfig.Complete().New()
	if err != nil {
		return err
	}
	return apiServer.GenericAPIServer.PrepareRun().RunWithContext(waitUtil.ContextForChannel(stopCh))
}
