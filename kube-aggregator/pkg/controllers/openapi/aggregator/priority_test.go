package aggregator

import (
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func newAPIServiceForTest(name, group string, minGroupPriority, versionPriority int32,
	svc *apiregistrationv1.ServiceReference) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: apiregistrationv1.APIServiceSpec{
			Group:                group,
			GroupPriorityMinimum: minGroupPriority,
			VersionPriority:      versionPriority,
			Service:              svc,
		},
	}
}

func assertSortedServices(t *testing.T, actual []*apiregistrationv1.APIService, expectedNames []string) {
	var actualNames []string
	for _, apiService := range actual {
		actualNames = append(actualNames, apiService.Name)
	}
	if !reflect.DeepEqual(actualNames, expectedNames) {
		t.Errorf("Expected %s got %s", expectedNames, actualNames)
	}
}

func TestAPIServiceSort(t *testing.T) {
	list := []*apiregistrationv1.APIService{
		newAPIServiceForTest("FirstService", "Group1", 10, 5, &apiregistrationv1.ServiceReference{}),
		newAPIServiceForTest("SecondService", "Group2", 15, 3, &apiregistrationv1.ServiceReference{}),
		newAPIServiceForTest("FirstServiceInternal", "Group1", 16, 3, &apiregistrationv1.ServiceReference{}),
		newAPIServiceForTest("ThirdService", "Group3", 15, 3, &apiregistrationv1.ServiceReference{}),
		newAPIServiceForTest("local_service_1", "Group4", 15, 1, nil),
		newAPIServiceForTest("local_service_3", "Group5", 15, 2, nil),
		newAPIServiceForTest("local_service_2", "Group6", 15, 3, nil),
	}
	sortByPriority(list)
	assertSortedServices(t, list, []string{"local_service_1", "local_service_2", "local_service_3", "FirstService", "FirstServiceInternal", "SecondService", "ThirdService"})
}
