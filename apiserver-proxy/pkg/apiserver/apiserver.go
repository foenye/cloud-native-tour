package apiserver

import (
	"context"
	"github.com/eonvon/cloud-native-tour/apiserver-proxy/pkg/proxy"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	apiserver "k8s.io/apiserver/pkg/server"
	clientGoKubeScheme "k8s.io/client-go/kubernetes/scheme"
	clientGoREST "k8s.io/client-go/rest"
	utilversion "k8s.io/component-base/version"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = clientGoKubeScheme.Scheme
	// Codecs provides methods for retrieving codecs and serializers for specific versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)
)

// ExtraConfig holds custom apiServer config
type ExtraConfig struct {
	REST *clientGoREST.Config
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *apiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

type completedConfig struct {
	GenericConfig apiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// ProxyAPIServer contains state for a kubernetes cluster master/api server.
type ProxyAPIServer struct {
	GenericAPIServer *apiserver.GenericAPIServer
}

// Complete fills in any fields not set that are required to hava valid data. It's mutating the receiver.
func (conf *Config) Complete() CompletedConfig {
	conf.GenericConfig.EffectiveVersion = utilversion.NewEffectiveVersion("1.0")
	completedConf := completedConfig{conf.GenericConfig.Complete(), &conf.ExtraConfig}

	// disable it so as to install our '/' handler
	completedConf.GenericConfig.EnableIndex = false
	// disable it so as forward to kube-apiserver
	completedConf.GenericConfig.EnableDiscovery = false

	completedConf.GenericConfig.EffectiveVersion = utilversion.NewEffectiveVersion("1.0")
	return CompletedConfig{&completedConf}
}

// NEW returns a new instance of WardleServer for given config.
func (completedConf *completedConfig) NEW() (*ProxyAPIServer, error) {
	genericAPIServer, err := completedConf.GenericConfig.New("kube-apiserver-proxy", apiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	proxyAPIServer := &ProxyAPIServer{GenericAPIServer: genericAPIServer}

	proxied, err := proxy.New(completedConf.ExtraConfig.REST, Codecs, Scheme)
	if err != nil {
		return nil, err
	}

	proxyAPIServer.GenericAPIServer.Handler.NonGoRestfulMux.HandlePrefix("/", proxied)
	proxyAPIServer.GenericAPIServer.AddPostStartHookOrDie("start-cache-informers", func(ctx apiserver.PostStartHookContext) error {
		ctxWithCancel, cancelFunc := context.WithCancel(context.Background())
		go func() {
			<-ctx.Done()
			cancelFunc()
		}()
		proxied.Start(ctxWithCancel)
		return nil
	})

	return proxyAPIServer, nil
}
