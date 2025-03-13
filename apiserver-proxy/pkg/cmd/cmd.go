package cmd

import (
	"flag"
	"fmt"
	"github.com/eonvon/cloud-native-tour/apiserver-proxy/pkg/apiserver"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server"
	serveroptions "k8s.io/apiserver/pkg/server/options"
	clientGoREST "k8s.io/client-go/rest"
	clientGoClientcmd "k8s.io/client-go/tools/clientcmd"
	cliFlag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	"net"
	"os"
)

type Options struct {
	SecureServing *serveroptions.SecureServingOptionsWithLoopback
	KubeConfig    string
	Features      *serveroptions.FeatureOptions

	Authentication *serveroptions.DelegatingAuthenticationOptions
	Authorization  *serveroptions.DelegatingAuthorizationOptions
}

func (opts *Options) Flags() (namedFlagSets cliFlag.NamedFlagSets) {
	flagSet := namedFlagSets.FlagSet("kube-apiserver-proxy")
	flagSet.StringVar(&opts.KubeConfig, "kubeconfig", opts.KubeConfig, "The path to the kubeconfig used to "+
		"connect to the Kubernetes API server (defaults to in-cluster config)")

	opts.SecureServing.AddFlags(namedFlagSets.FlagSet("apiserver secure serving"))
	opts.Features.AddFlags(namedFlagSets.FlagSet("features"))

	opts.Authentication.AddFlags(namedFlagSets.FlagSet("apiserver authentication"))
	opts.Authorization.AddFlags(namedFlagSets.FlagSet("apiserver authorization"))
	return namedFlagSets
}

// Complete fills in fields required to hava valid data
func (opts *Options) Complete() error {
	return nil
}

// Validate serverOptions
func (opts *Options) Validate(_ []string) error {
	var errs []error
	errs = append(errs, opts.Authentication.Validate()...)
	errs = append(errs, opts.Authorization.Validate()...)
	return errors.NewAggregate(errs)
}

func (opts *Options) APIServerConfig() (*server.RecommendedConfig, error) {
	if err := opts.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil,
		[]net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	apiServerConfig := server.NewRecommendedConfig(apiserver.Codecs)
	if err := opts.SecureServing.ApplyTo(&apiServerConfig.SecureServing, &apiServerConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	if err := opts.Authentication.ApplyTo(&apiServerConfig.Authentication, apiServerConfig.SecureServing,
		nil); err != nil {
		return nil, err
	}
	if err := opts.Authorization.ApplyTo(&apiServerConfig.Authorization); err != nil {
		return nil, err
	}
	return apiServerConfig, nil
}

func (opts *Options) restConfig() (*clientGoREST.Config, error) {
	var clientGoRESTConfig *clientGoREST.Config
	var err error
	if len(opts.KubeConfig) > 0 {
		loadingRules := &clientGoClientcmd.ClientConfigLoadingRules{ExplicitPath: opts.KubeConfig}
		clientConfigLoader := clientGoClientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientGoClientcmd.ConfigOverrides{})

		clientGoRESTConfig, err = clientConfigLoader.ClientConfig()
	} else {
		clientGoRESTConfig, err = clientGoREST.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("unable to construct lister client config: %v", err)
	}

	// use protobuf protocol for communication with apiserver.
	clientGoRESTConfig.ContentType = "application/vnd.kubernetes.protobuf"
	_ = clientGoREST.SetKubernetesDefaults(clientGoRESTConfig)
	return clientGoRESTConfig, nil
}

func (opts *Options) ServerConfig() (*apiserver.Config, error) {
	apiServerConfig, err := opts.APIServerConfig()
	if err != nil {
		return nil, err
	}
	restConfig, err := opts.restConfig()
	if err != nil {
		return nil, err
	}
	return &apiserver.Config{
		GenericConfig: apiServerConfig,
		ExtraConfig: apiserver.ExtraConfig{
			REST: restConfig,
		},
	}, nil
}

func runCommand(options *Options, stopChan <-chan struct{}) error {
	serverConfig, err := options.ServerConfig()
	if err != nil {
		return err
	}

	proxyAPIServer, err := serverConfig.Complete().NEW()
	if err != nil {
		return err
	}
	return proxyAPIServer.GenericAPIServer.PrepareRun().RunWithContext(utilwait.ContextForChannel(stopChan))
}

func NewProxyServerCommand(stopChan <-chan struct{}) *cobra.Command {
	opts := &Options{
		SecureServing:  serveroptions.NewSecureServingOptions().WithLoopback(),
		Authentication: serveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:  serveroptions.NewDelegatingAuthorizationOptions(),
	}

	command := &cobra.Command{
		Short: "Launch kube-apiserver-proxy",
		Long:  "Launch kube-apiserver-proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(); err != nil {
				return err
			}
			if err := opts.Validate(args); err != nil {
				return err
			}
			if err := runCommand(opts, stopChan); err != nil {
				return err
			}
			return nil
		},
	}

	flags := command.Flags()
	optFlags := opts.Flags()
	for _, flagSet := range optFlags.FlagSets {
		flags.AddFlagSet(flagSet)
	}
	local := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(local)
	optFlags.FlagSet("logging").AddGoFlagSet(local)

	usageFmt := "Usage:\n  %s\n"
	columns, _, _ := term.TerminalSize(command.OutOrStdout())
	command.SetUsageFunc(func(cmd *cobra.Command) error {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), usageFmt, cmd.UseLine())
		cliFlag.PrintSections(cmd.OutOrStderr(), optFlags, columns)
		return nil
	})
	command.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliFlag.PrintSections(cmd.OutOrStdout(), optFlags, columns)
	})

	return command
}
