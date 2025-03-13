package apiserver

import (
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io"
	helloInstaller "github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io/install"
	transformationInstaller "github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/transformation/install"
	registryFoo "github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/registry/hello.eonvon.github.io/foo"
	hellov1 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v1"
	hellov2 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	clientGoREST "k8s.io/client-go/rest"
	utilversion "k8s.io/component-base/version"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codes provides methods for retrieving codecs and serializers for specific versions and content types.
	Codes = serializer.NewCodecFactory(Scheme)
)

func init() {
	helloInstaller.Install(Scheme)
	transformationInstaller.Install(Scheme)

	// we need to add the options to empty v1
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Group: "", Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	REST              *clientGoREST.Config
	EnableEtcdStorage bool
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *server.RecommendedConfig
	ExtraConfig   ExtraConfig
}

func (config *Config) Complete() CompletedConfig {
	config.GenericConfig.EffectiveVersion = utilversion.NewEffectiveVersion("1.0")
	completedConf := completedConfig{GenericConfig: config.GenericConfig.Complete(), ExtraConfig: &config.ExtraConfig}
	return CompletedConfig{&completedConf}
}

// HelloAPIServer contains state for a Kubernetes cluster master/api server.
type HelloAPIServer struct {
	GenericAPIServer *server.GenericAPIServer
}

type completedConfig struct {
	GenericConfig server.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

func (completedCfg *completedConfig) New() (*HelloAPIServer, error) {
	genericAPIServer, err := completedCfg.GenericConfig.New(hello.Group+"-apiserver", server.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}
	helloAPIServer := &HelloAPIServer{GenericAPIServer: genericAPIServer}

	defaultAPIGroupInfo := server.NewDefaultAPIGroupInfo(hello.Group, Scheme, metav1.ParameterCodec, Codes)

	if !completedCfg.ExtraConfig.EnableEtcdStorage {
		inMemory := registryFoo.NewInMemory()
		v1Storage := map[string]rest.Storage{hello.Plural: inMemory}
		v2Storage := map[string]rest.Storage{hello.Plural: inMemory}
		defaultAPIGroupInfo.VersionedResourcesStorageMap[hellov1.Version] = v1Storage
		defaultAPIGroupInfo.VersionedResourcesStorageMap[hellov2.Version] = v2Storage
	} else {
		etcd, err := registryFoo.NewREST(Scheme, completedCfg.GenericConfig.RESTOptionsGetter)
		if err != nil {
			return nil, err
		}
		v1Storage := map[string]rest.Storage{hello.Plural: etcd.Foo, hello.Plural + "/base64": etcd.Base64}
		v2Storage := map[string]rest.Storage{
			hello.Plural:             etcd.Foo,
			hello.Plural + "/config": etcd.Config,
			hello.Plural + "/status": etcd.Status,
			hello.Plural + "/base64": etcd.Base64,
		}
		defaultAPIGroupInfo.VersionedResourcesStorageMap[hellov1.Version] = v1Storage
		defaultAPIGroupInfo.VersionedResourcesStorageMap[hellov2.Version] = v2Storage

	}
	if err := helloAPIServer.GenericAPIServer.InstallAPIGroup(&defaultAPIGroupInfo); err != nil {
		return nil, err
	}

	return helloAPIServer, nil
}
