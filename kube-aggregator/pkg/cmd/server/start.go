package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiregistrationv1beta1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apiserver"
	apiserverSchema "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apiserver/scheme"
	generatedOpenapi "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/generated/openapi"
	"io"
	utilErrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	endpointsOpenAPI "k8s.io/apiserver/pkg/endpoints/openapi"
	genericAPIServer "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/component-base/featuregate"
)

// NewCommandStartAggregator provides a CLI handler for 'start master' command with a default AggregatorOptions.
func NewCommandStartAggregator(ctx context.Context, defaultOptions *AggregatorOptions) *cobra.Command {
	aggregatorOptions := *defaultOptions
	cmd := &cobra.Command{
		Short: "Launch a API aggregator and proxy server",
		Long:  "Launch a API aggregator and proxy server",

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return featuregate.DefaultComponentGlobalsRegistry.Set()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aggregatorOptions.Complete(); err != nil {
				return err
			}
			if err := aggregatorOptions.Validate(args); err != nil {
				return err
			}
			if err := aggregatorOptions.RunAggregator(cmd.Context()); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.SetContext(ctx)

	aggregatorOptions.AddFlags(cmd.Flags())
	return cmd
}

const defaultEtcdPathPrefix = "/registry/kube-aggregator.kubernetes.io/"

type aggregatorOptionsImplementation interface {
	AddFlags(flagSet *pflag.FlagSet)
	Validate(args []string) error
	Complete() error
	RunAggregator(ctx context.Context) error
}

var _ aggregatorOptionsImplementation = &AggregatorOptions{}

// AggregatorOptions contains everything necessary to create and run an API Aggregator.
type AggregatorOptions struct {
	ServerRunOptions     *options.ServerRunOptions
	RecommendedOptions   *options.RecommendedOptions
	APIEnablementOptions *options.APIEnablementOptions

	// ProxyClientCert/Key are the client cert used to identify this proxy. Backing APIServices use
	// this to confirm the proxy's identity.
	ProxyClientCertFile string
	ProxyClientKeyFile  string

	StdOut io.Writer
	StdErr io.Writer
}

// NewDefaultOptions builds a "normal" set of options.  You wouldn't normally expose this, but hyper-kube isn't
// cobra compatible.
func NewDefaultOptions(out, err io.Writer) *AggregatorOptions {
	return &AggregatorOptions{
		ServerRunOptions:     options.NewServerRunOptions(),
		RecommendedOptions:   options.NewRecommendedOptions(defaultEtcdPathPrefix, apiserverSchema.Codecs.LegacyCodec(apiregistrationv1beta1.SchemeGroupVersion)),
		APIEnablementOptions: options.NewAPIEnablementOptions(),
		StdOut:               out,
		StdErr:               err,
	}
}

// AddFlags is necessary because hyper-kube doesn't work using cobra, so we hava to different registration and
// execution paths.
func (options *AggregatorOptions) AddFlags(flagSet *pflag.FlagSet) {
	options.ServerRunOptions.AddUniversalFlags(flagSet)
	options.RecommendedOptions.AddFlags(flagSet)
	options.APIEnablementOptions.AddFlags(flagSet)

	flagSet.StringVar(&options.ProxyClientCertFile, "proxy-client-cert-file", options.ProxyClientCertFile,
		"Client certificate used identity the proxy to the API server")
	flagSet.StringVar(&options.ProxyClientKeyFile, "proxy-client-key-file", options.ProxyClientKeyFile,
		"Client certificate key used identity the proxy to the API server")
}

// Validate validates all required options.
func (options *AggregatorOptions) Validate(_ []string) error {
	var errorList []error
	errorList = append(errorList, options.ServerRunOptions.Validate()...)
	errorList = append(errorList, options.RecommendedOptions.Validate()...)
	errorList = append(errorList, options.APIEnablementOptions.Validate(apiserverSchema.Scheme)...)
	return utilErrors.NewAggregate(errorList)
}

// Complete fills in missing options.
func (options *AggregatorOptions) Complete() error {
	return options.ServerRunOptions.Complete()
}

// RunAggregator runs the API Aggregator.
func (options *AggregatorOptions) RunAggregator(ctx context.Context) error {
	// TODO have a "real" external address
	if err := options.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost",
		nil, nil); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	recommendedConfig := genericAPIServer.NewRecommendedConfig(apiserverSchema.Codecs)

	if err := options.ServerRunOptions.ApplyTo(&recommendedConfig.Config); err != nil {
		return err
	}
	if err := options.RecommendedOptions.ApplyTo(recommendedConfig); err != nil {
		return err
	}
	if err := options.APIEnablementOptions.ApplyTo(&recommendedConfig.Config, apiserver.DefaultAPIResourceConfigSource(),
		apiserverSchema.Scheme); err != nil {
		return err
	}
	recommendedConfig.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("attach", "exec", "proxy", "log", "portforward"),
	)
	recommendedConfig.OpenAPIConfig = genericAPIServer.DefaultOpenAPIConfig(generatedOpenapi.GetOpenAPIDefinitions,
		endpointsOpenAPI.NewDefinitionNamer(apiserverSchema.Scheme))
	recommendedConfig.OpenAPIConfig.Info.Title = "kube-aggregator"
	// prevent generic API server from installing the OpenAPI handler. Aggregator server
	// has its own customized OpenAPI handler.
	recommendedConfig.SkipOpenAPIInstallation = true

	serviceResolver := apiserver.NewClusterIPServiceResolver(recommendedConfig.SharedInformerFactory.Core().V1().
		Services().Lister())

	config := apiserver.Config{GenericConfig: recommendedConfig, ExtraConfig: apiserver.ExtraConfig{
		ServiceResolver: serviceResolver,
	}}

	if len(options.ProxyClientCertFile) == 0 || len(options.ProxyClientKeyFile) == 0 {
		return errors.New("missing a client certificate along with a key to identify the proxy to the API Server")
	}

	config.ExtraConfig.ProxyClientCertFile = options.ProxyClientCertFile
	config.ExtraConfig.ProxyClientKeyFile = options.ProxyClientKeyFile

	delegate, err := config.Complete().NewWithDelegate(genericAPIServer.NewEmptyDelegate())
	if err != nil {
		return err
	}

	prepared, err := delegate.PreparedRun()
	if err != nil {
		return err
	}

	return prepared.Run(ctx)
}
