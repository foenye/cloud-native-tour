package autoregister

import (
	"fmt"
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset/fake"
	generatedClientListersV1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientGoTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sync"
	"testing"
)

func newAutoRegisterManagedAPIService(name string) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{ManagedLabel: manageContinuously}},
	}
}

func newAutoRegisterManagedOnStartAPIService(name string) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{ManagedLabel: manageOnStart}},
	}
}

func newAutoRegisterManagedModifiedAPIService(name string) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{ManagedLabel: manageContinuously}},
		Spec:       apiregistrationv1.APIServiceSpec{Group: "something"},
	}
}

func newAPIService(name string) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func checkForNothing(_ string, // name
	client *fake.Clientset) error {
	if len(client.Actions()) > 0 {
		return fmt.Errorf("unexpected action: %v", client.Actions())
	}
	return nil
}

func checkForCreate(name string, client *fake.Clientset) error {
	if len(client.Actions()) == 0 {
		return nil
	}
	if len(client.Actions()) > 1 {
		return fmt.Errorf("unexpected action: %v", client.Actions())
	}

	action := client.Actions()[0]
	createAction, casted := action.(clientGoTesting.CreateAction)
	if !casted {
		return fmt.Errorf("unexpected action: %v", client.Actions())
	}
	apiService := createAction.GetObject().(*apiregistrationv1.APIService)
	if apiService.Name != name || apiService.Labels[ManagedLabel] != manageContinuously {
		return fmt.Errorf("bad name or label %v", createAction)
	}
	return nil
}

func checkForCreateOnStart(name string, client *fake.Clientset) error {
	if len(client.Actions()) == 0 {
		return nil
	}
	if len(client.Actions()) > 1 {
		return fmt.Errorf("unexpected action: %v", client.Actions())
	}

	action := client.Actions()[0]
	createAction, casted := action.(clientGoTesting.CreateAction)
	if !casted {
		return fmt.Errorf("unexpected action: %v", client.Actions())
	}
	apiService := createAction.GetObject().(*apiregistrationv1.APIService)
	if apiService.Name != name || apiService.Labels[ManagedLabel] != manageOnStart {
		return fmt.Errorf("bad name or label %v", createAction)
	}
	return nil
}

func checkForUpdate(name string, client *fake.Clientset) error {
	if len(client.Actions()) == 0 {
		return nil
	}
	if len(client.Actions()) > 1 {
		return fmt.Errorf("unexpected action: %v", client.Actions())
	}

	action := client.Actions()[0]
	updateAction, casted := action.(clientGoTesting.UpdateAction)
	if !casted {
		return fmt.Errorf("unexpected action: %v", client.Actions())
	}
	apiService := updateAction.GetObject().(*apiregistrationv1.APIService)
	if apiService.Name != name || apiService.Labels[ManagedLabel] != manageContinuously || apiService.Spec.Group != "" {
		return fmt.Errorf("bad name, label, or group %v", updateAction)
	}
	return nil
}

func checkForDelete(name string, client *fake.Clientset) error {
	if len(client.Actions()) == 0 {
		return nil
	}

	for _, action := range client.Actions() {
		deleteAction, casted := action.(clientGoTesting.DeleteAction)
		if !casted {
			return fmt.Errorf("unexpected action: %v", client.Actions())
		}
		if deleteAction.GetName() != name {
			return fmt.Errorf("bad name %v", deleteAction)
		}
	}
	return nil
}

func TestSync(t *testing.T) {
	tests := []struct {
		name                      string
		apiServiceName            string
		addAPIServices            []*apiregistrationv1.APIService
		updateAPIServices         []*apiregistrationv1.APIService
		addSyncAPIServices        []*apiregistrationv1.APIService
		addSyncOnStartAPIServices []*apiregistrationv1.APIService
		delSyncAPIServices        []string
		alreadySynced             map[string]bool
		presentAtStart            map[string]bool
		expectedResults           func(name string, client *fake.Clientset) error
	}{
		{
			name:               "adding an API service which isn't auto-managed does nothing",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{newAPIService("foo")},
			updateAPIServices:  []*apiregistrationv1.APIService{},
			addSyncAPIServices: []*apiregistrationv1.APIService{},
			delSyncAPIServices: []string{},
			expectedResults:    checkForNothing,
		},
		{
			name:               "adding one to auto-register should create",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{},
			updateAPIServices:  []*apiregistrationv1.APIService{},
			addSyncAPIServices: []*apiregistrationv1.APIService{newAPIService("foo")},
			delSyncAPIServices: []string{},
			expectedResults:    checkForCreate,
		},
		{
			name:               "duplicate AddAPIServiceToSync don't panic",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{newAutoRegisterManagedAPIService("foo")},
			updateAPIServices:  []*apiregistrationv1.APIService{},
			addSyncAPIServices: []*apiregistrationv1.APIService{newAutoRegisterManagedAPIService("foo"), newAutoRegisterManagedAPIService("foo")},
			delSyncAPIServices: []string{},
			expectedResults:    checkForNothing,
		},
		{
			name:               "duplicate RemoveAPIServiceToSync don't panic",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{newAutoRegisterManagedAPIService("foo")},
			updateAPIServices:  []*apiregistrationv1.APIService{},
			addSyncAPIServices: []*apiregistrationv1.APIService{},
			delSyncAPIServices: []string{"foo", "foo"},
			expectedResults:    checkForDelete,
		},
		{
			name:               "removing auto-managed then RemoveAPIService should not touch APIService",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{},
			updateAPIServices:  []*apiregistrationv1.APIService{newAPIService("foo")},
			addSyncAPIServices: []*apiregistrationv1.APIService{},
			delSyncAPIServices: []string{"foo"},
			expectedResults:    checkForNothing,
		},
		{
			name:               "create managed apiservice without a matching request",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{newAPIService("foo")},
			updateAPIServices:  []*apiregistrationv1.APIService{newAutoRegisterManagedAPIService("foo")},
			addSyncAPIServices: []*apiregistrationv1.APIService{},
			delSyncAPIServices: []string{},
			expectedResults:    checkForDelete,
		},
		{
			name:               "modifying it should result in stomping",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{},
			updateAPIServices:  []*apiregistrationv1.APIService{newAutoRegisterManagedModifiedAPIService("foo")},
			addSyncAPIServices: []*apiregistrationv1.APIService{newAutoRegisterManagedAPIService("foo")},
			delSyncAPIServices: []string{},
			expectedResults:    checkForUpdate,
		},

		{
			name:                      "adding one to auto-register on start should create",
			apiServiceName:            "foo",
			addAPIServices:            []*apiregistrationv1.APIService{},
			updateAPIServices:         []*apiregistrationv1.APIService{},
			addSyncOnStartAPIServices: []*apiregistrationv1.APIService{newAPIService("foo")},
			delSyncAPIServices:        []string{},
			expectedResults:           checkForCreateOnStart,
		},
		{
			name:                      "adding one to auto-register on start already synced should do nothing",
			apiServiceName:            "foo",
			addAPIServices:            []*apiregistrationv1.APIService{},
			updateAPIServices:         []*apiregistrationv1.APIService{},
			addSyncOnStartAPIServices: []*apiregistrationv1.APIService{newAPIService("foo")},
			delSyncAPIServices:        []string{},
			alreadySynced:             map[string]bool{"foo": true},
			expectedResults:           checkForNothing,
		},
		{
			name:               "managed onstart apiservice present at start without a matching request should delete",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{newAPIService("foo")},
			updateAPIServices:  []*apiregistrationv1.APIService{newAutoRegisterManagedOnStartAPIService("foo")},
			addSyncAPIServices: []*apiregistrationv1.APIService{},
			delSyncAPIServices: []string{},
			presentAtStart:     map[string]bool{"foo": true},
			alreadySynced:      map[string]bool{},
			expectedResults:    checkForDelete,
		},
		{
			name:               "managed onstart apiservice present at start without a matching request already synced once should no-op",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{newAPIService("foo")},
			updateAPIServices:  []*apiregistrationv1.APIService{newAutoRegisterManagedOnStartAPIService("foo")},
			addSyncAPIServices: []*apiregistrationv1.APIService{},
			delSyncAPIServices: []string{},
			presentAtStart:     map[string]bool{"foo": true},
			alreadySynced:      map[string]bool{"foo": true},
			expectedResults:    checkForNothing,
		},
		{
			name:               "managed onstart apiservice not present at start without a matching request should no-op",
			apiServiceName:     "foo",
			addAPIServices:     []*apiregistrationv1.APIService{newAPIService("foo")},
			updateAPIServices:  []*apiregistrationv1.APIService{newAutoRegisterManagedOnStartAPIService("foo")},
			addSyncAPIServices: []*apiregistrationv1.APIService{},
			delSyncAPIServices: []string{},
			presentAtStart:     map[string]bool{},
			alreadySynced:      map[string]bool{},
			expectedResults:    checkForNothing,
		},
		{
			name:                      "modifying onstart it should result in stomping",
			apiServiceName:            "foo",
			addAPIServices:            []*apiregistrationv1.APIService{},
			updateAPIServices:         []*apiregistrationv1.APIService{newAutoRegisterManagedModifiedAPIService("foo")},
			addSyncOnStartAPIServices: []*apiregistrationv1.APIService{newAutoRegisterManagedOnStartAPIService("foo")},
			delSyncAPIServices:        []string{},
			expectedResults:           checkForUpdate,
		},
		{
			name:                      "modifying onstart already synced should no-op",
			apiServiceName:            "foo",
			addAPIServices:            []*apiregistrationv1.APIService{},
			updateAPIServices:         []*apiregistrationv1.APIService{newAutoRegisterManagedModifiedAPIService("foo")},
			addSyncOnStartAPIServices: []*apiregistrationv1.APIService{newAutoRegisterManagedOnStartAPIService("foo")},
			delSyncAPIServices:        []string{},
			alreadySynced:             map[string]bool{"foo": true},
			expectedResults:           checkForNothing,
		},
	}

	for _, test := range tests {
		//goland:noinspection GoDeprecation
		fakeClient := fake.NewSimpleClientset()
		apiServiceIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

		alreadySynced := map[string]bool{}
		for key, val := range test.alreadySynced {
			alreadySynced[key] = val
		}

		presentAtStart := map[string]bool{}
		for key, val := range test.presentAtStart {
			presentAtStart[key] = val
		}

		ctrl := &controller{
			apiServiceClient:  fakeClient.ApiregistrationV1(),
			apiServiceLister:  generatedClientListersV1.NewAPIServiceLister(apiServiceIndexer),
			apiServicesToSync: map[string]*apiregistrationv1.APIService{},
			queue: workqueue.NewTypedRateLimitingQueueWithConfig(
				workqueue.DefaultTypedControllerRateLimiter[string](),
				workqueue.TypedRateLimitingQueueConfig[string]{Name: "autoregister"},
			),

			syncedSuccessfullyLock: &sync.RWMutex{},
			syncedSuccessfully:     alreadySynced,

			apiServicesAtStart: presentAtStart,
		}

		for _, obj := range test.addAPIServices {
			_ = apiServiceIndexer.Add(obj)
		}
		for _, obj := range test.updateAPIServices {
			_ = apiServiceIndexer.Update(obj)
		}
		for _, obj := range test.addSyncAPIServices {
			ctrl.AddAPIServiceToSync(obj)
		}
		for _, obj := range test.addSyncOnStartAPIServices {
			ctrl.AddAPIServiceToSyncOnStart(obj)
		}
		for _, obj := range test.delSyncAPIServices {
			ctrl.RemoveAPIServiceToSync(obj)
		}

		_ = ctrl.checkAPIService(test.apiServiceName)
		// compare the expected results
		if err := test.expectedResults(test.apiServiceName, fakeClient); err != nil {
			t.Errorf("%s %v", test.name, err)
		}
	}
}
