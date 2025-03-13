package apiserver

import (
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-v1/pkg/registry/hello.eonvon.github.io/foo"
	hello "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io"
	hellov1 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericAPIServer "k8s.io/apiserver/pkg/server"
	versionUtils "k8s.io/apiserver/pkg/util/version"
	clientGoREST "k8s.io/client-go/rest"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()

	// Codecs provides methods for retrieving codecs and serializers for specific versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	hello.Install(Scheme)

	// we need to add the options to empty v1
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Group: "", Version: hellov1.Version})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: hellov1.Version}
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
	Rest              *clientGoREST.Config
	EnableEtcdStorage bool
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *genericAPIServer.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// HelloAPIServer contains state for a Kubernetes cluster master/api server.
type HelloAPIServer struct {
	GenericAPIServer *genericAPIServer.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericAPIServer.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to hava valid data. It's mutating the receiver.
func (config *Config) Complete() CompletedConfig {
	genericConfig := config.GenericConfig
	genericConfig.EffectiveVersion = versionUtils.NewEffectiveVersion("1.0")
	completedCfg := completedConfig{
		genericConfig.Complete(),
		&config.ExtraConfig,
	}

	return CompletedConfig{&completedCfg}
}

func (fromConfig completedConfig) New() (*HelloAPIServer, error) {
	genericServer, err := fromConfig.GenericConfig.New(hellov1.Group+"-apiserver", genericAPIServer.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	helloAPIServer := &HelloAPIServer{GenericAPIServer: genericServer}

	apiGroupInfo := genericAPIServer.NewDefaultAPIGroupInfo(hellov1.Group, Scheme, runtime.NewParameterCodec(Scheme), Codecs)

	apiGroupInfo.VersionedResourcesStorageMap[hellov1.Version] = map[string]rest.Storage{}
	if fromConfig.ExtraConfig.EnableEtcdStorage {
		if etcdStorage, err := foo.NewREST(Scheme, fromConfig.GenericConfig.RESTOptionsGetter); err != nil {
			return nil, err
		} else {
			apiGroupInfo.VersionedResourcesStorageMap[hellov1.Version][hellov1.Plural] = etcdStorage
		}
	} else {
		apiGroupInfo.VersionedResourcesStorageMap[hellov1.Version][hellov1.Plural] = foo.NewEmbeddedStore()
	}

	if err := helloAPIServer.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}
	return helloAPIServer, nil
}
