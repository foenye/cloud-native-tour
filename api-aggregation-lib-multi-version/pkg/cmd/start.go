package cmd

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/admisssion/disallow"
	"github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.yeahfo.github.io"
	hellov1 "github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.yeahfo.github.io/v1"
	customapiserver "github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/apiserver"
	generatedopenapi "github.com/yeahfo/cloud-native-tour/api/generated/openapi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	serveroptions "k8s.io/apiserver/pkg/server/options"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	"net"
	"os"
)

const defaultEtcdPathPrefix = "/registry/" + hello.Group

type Options struct {
	SecureServing *serveroptions.SecureServingOptionsWithLoopback
	Kubeconfig    string
	Features      *serveroptions.FeatureOptions

	EnableEtcdStorage bool
	Etcd              *serveroptions.EtcdOptions

	EnableAuth     bool
	Authentication *serveroptions.DelegatingAuthenticationOptions
	Authorization  *serveroptions.DelegatingAuthorizationOptions

	EnableAdmission bool
	Admission       *serveroptions.AdmissionOptions
}

func (options *Options) Flags() (flagSets cliflag.NamedFlagSets) {
	servingFlagSet := flagSets.FlagSet(hello.GroupName + "-server")
	servingFlagSet.StringVar(&options.Kubeconfig, "kubeconfig", options.Kubeconfig, "The path to the "+
		"kubeconfig used to connect to the Kubernetes API server (default to in-cluster config)")

	options.SecureServing.AddFlags(flagSets.FlagSet("apiserver secure serving"))
	options.Features.AddFlags(flagSets.FlagSet("features"))

	servingFlagSet.BoolVar(&options.EnableEtcdStorage, "enable-etcd-storage", false, "If true, "+
		"storage object in etcd")
	options.Features.AddFlags(flagSets.FlagSet("Etcd"))

	servingFlagSet.BoolVar(&options.EnableAuth, "enable-auth", options.EnableAuth, "If true, enable "+
		"authn and authz")
	options.Authentication.AddFlags(flagSets.FlagSet("apiserver authentication"))
	options.Authorization.AddFlags(flagSets.FlagSet("apiserver authorization"))

	servingFlagSet.BoolVar(&options.EnableAdmission, "enable-admission", options.EnableAdmission, "If "+
		"true, enable admission plugins")

	return flagSets
}

// Complete fills in fields required to hava valid data.
func (options *Options) Complete() error {
	disallow.Register(options.Admission.Plugins)
	options.Admission.RecommendedPluginOrder = append(options.Admission.RecommendedPluginOrder, disallow.PluginDisallowFoo)
	return nil
}

// Validation ServerOptions
func (options *Options) Validation(_ []string) error {
	var errs []error

	if options.EnableEtcdStorage {
		errs = options.Etcd.Validate()
	}

	if options.EnableAuth {
		errs = append(errs, options.Authentication.Validate()...)
		errs = append(errs, options.Authorization.Validate()...)
	}
	return errors.NewAggregate(errs)
}

func (options *Options) APIServerConfig() (*server.RecommendedConfig, error) {
	if err := options.SecureServing.MaybeDefaultWithSelfSignedCerts("loccalhost", nil, []net.IP{
		net.ParseIP("127.0.0.1"),
	}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	recommendedConfig := server.NewRecommendedConfig(customapiserver.Codes)
	if err := options.SecureServing.ApplyTo(&recommendedConfig.SecureServing, &recommendedConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	// enable OpenAPI schemas
	definitionNamer := openapi.NewDefinitionNamer(customapiserver.Scheme)
	recommendedConfig.OpenAPIConfig = server.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, definitionNamer)
	recommendedConfig.OpenAPIConfig.Info.Title = hello.Group + "-server"
	recommendedConfig.OpenAPIConfig.Info.Version = "0.0.1"
	recommendedConfig.OpenAPIV3Config = server.DefaultOpenAPIV3Config(generatedopenapi.GetOpenAPIDefinitions, definitionNamer)
	recommendedConfig.OpenAPIV3Config.Info.Title = hello.Group + "-server"
	recommendedConfig.OpenAPIV3Config.Info.Version = "0.0.1"

	if options.EnableAuth {
		if err := options.Authentication.ApplyTo(&recommendedConfig.Authentication, recommendedConfig.SecureServing,
			nil); err != nil {
			return nil, err
		}
		if err := options.Authorization.ApplyTo(&recommendedConfig.Authorization); err != nil {
			return nil, err
		}
	}

	if options.EnableAdmission {
		if err := (&serveroptions.CoreAPIOptions{}).ApplyTo(recommendedConfig); err != nil {
			return nil, err
		}
		kubeClient, err := kubernetes.NewForConfig(recommendedConfig.ClientConfig)
		if err != nil {
			return nil, err
		}
		dynamicClient, err := dynamic.NewForConfig(recommendedConfig.ClientConfig)
		if err != nil {
			return nil, err
		}
		var pluginInitializers []admission.PluginInitializer
		if err := options.Admission.ApplyTo(&recommendedConfig.Config, recommendedConfig.SharedInformerFactory, kubeClient,
			dynamicClient, feature.DefaultFeatureGate, pluginInitializers...); err != nil {
			return nil, err
		}
	}
	return recommendedConfig, nil
}

func (options *Options) ServerConfig() (*customapiserver.Config, error) {
	apiServerConfig, err := options.APIServerConfig()
	if err != nil {
		return nil, err
	}

	if options.EnableEtcdStorage {
		if storageConfigCopied := options.Etcd.StorageConfig; storageConfigCopied.StorageObjectCountTracker == nil {
			storageConfigCopied.StorageObjectCountTracker = apiServerConfig.StorageObjectCountTracker
		}
		klog.Infof("Etcd config: %v", options.Etcd)
		// set apiservercfg's RESTOptionsGetter as StorageFactoryRestOptionsFactory{..., StorageFactory: DefaultStorageFactory}
		// like https://github.com/kubernetes/kubernetes/blob/e1ad9bee5bba8fbe85a6bf6201379ce8b1a611b1/cmd/kube-apiserver/app/server.go#L407-L415
		// DefaultStorageFactory#NewConfig provides a way to negotiate StorageSerializer/DeSerializer by Etcd.DefaultStorageMediaType option
		//
		// DefaultStorageFactory's NewConfig will be called by interface genericregistry.RESTOptionsGetter#GetRESTOptions (struct StorageFactoryRestOptionsFactory)
		// interface genericregistry.RESTOptionsGetter#GetRESTOptions will be called by genericregistry.Store#CompleteWithOptions
		// Finally all RESTBackend Options will be passed to genericregistry.Store implementations
		if err := options.Etcd.ApplyWithStorageFactoryTo(serverstorage.NewDefaultStorageFactory(
			options.Etcd.StorageConfig,
			options.Etcd.DefaultStorageMediaType,
			customapiserver.Codes,
			serverstorage.NewDefaultResourceEncodingConfig(customapiserver.Scheme),
			apiServerConfig.MergedResourceConfig,
			nil), &apiServerConfig.Config); err != nil {
			return nil, err
		}
	}

	return &customapiserver.Config{GenericConfig: apiServerConfig, ExtraConfig: customapiserver.ExtraConfig{
		EnableEtcdStorage: options.EnableEtcdStorage,
	}}, nil
}

func NewHelloServerCommand(stopCh <-chan struct{}) *cobra.Command {
	options := &Options{
		SecureServing: serveroptions.NewSecureServingOptions().WithLoopback(),
		// if just encode as json and store to etcd, just do this
		// Etcd: genericoptions.NewEtcdOptions(storagebackend.NewDefaultConfig(defaultEtcdPathPrefix, myapiserver.Codecs.LegacyCodec(hellov1.SchemeGroupVersion))),
		// but if we want to encode as json and pb, just assign nil to Codec here
		// like the official kube-apiserver https://github.com/kubernetes/kubernetes/blob/e1ad9bee5bba8fbe85a6bf6201379ce8b1a611b1/cmd/kube-apiserver/app/options/options.go#L96
		// when new/complete apiserver config, use EtcdOptions#ApplyWithStorageFactoryTo server.Config, which
		// finally init server.Config.RESTOptionsGetter as StorageFactoryRestOptionsFactory
		Etcd:           serveroptions.NewEtcdOptions(storagebackend.NewDefaultConfig(defaultEtcdPathPrefix, nil)),
		Authentication: serveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:  serveroptions.NewDelegatingAuthorizationOptions(),
		Admission:      serveroptions.NewAdmissionOptions(),
	}
	options.Etcd.StorageConfig.EncodeVersioner = runtime.NewMultiGroupVersioner(hellov1.SchemeGroupVersion,
		schema.GroupKind{Group: hellov1.SchemeGroupVersion.Group})
	options.Etcd.DefaultStorageMediaType = "application/json"
	options.SecureServing.BindPort = 6443

	command := &cobra.Command{
		Short: "Launch " + hello.Group + "-server",
		Long:  "Launch " + hello.Group + "-server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.Complete(); err != nil {
				return err
			}
			if err := options.Validation(args); err != nil {
				return err
			}
			if err := runCommand(options, stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := command.Flags()
	optFlags := options.Flags()
	for _, flagSet := range optFlags.FlagSets {
		flags.AddFlagSet(flagSet)
	}
	osFlag := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(osFlag)
	optFlags.FlagSet("logging").AddGoFlagSet(osFlag)

	usageFmt := "Usage:\n  %s\n"
	columns, _, _ := term.TerminalSize(command.OutOrStdout())
	command.SetUsageFunc(func(cmd *cobra.Command) error {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), optFlags, columns)
		return nil
	})
	command.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), optFlags, columns)
	})

	return command
}

func runCommand(options *Options, stopCh <-chan struct{}) error {
	serverConfig, err := options.ServerConfig()
	if err != nil {
		return err
	}
	apiServer, err := serverConfig.Complete().New()
	if err != nil {
		return err
	}
	return apiServer.GenericAPIServer.PrepareRun().RunWithContext(utilwait.ContextForChannel(stopCh))
}
