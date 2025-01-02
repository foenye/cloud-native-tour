package helper

import (
	v1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"testing"
)

var (
	a v1.APIServiceConditionType = "A"
	b v1.APIServiceConditionType = "B"
)

func TestIsAPIServiceConditionTrue(t *testing.T) {
	conditionATrue := makeNewAPIServiceCondition(a, "a reason", "a message", v1.ConditionTrue)
	conditionAFalse := makeNewAPIServiceCondition(a, "a reason", "a message", v1.ConditionFalse)
	testCases := []*struct {
		name          string
		apiService    *v1.APIService
		conditionType v1.APIServiceConditionType
		expected      bool
	}{
		{
			name:          "Should return false when condition of type not present",
			apiService:    makeNewAPIService("v1", 100),
			conditionType: a,
			expected:      false,
		},
		{
			name:          "Should return false when condition of type is present but status is not ConditionTrue",
			apiService:    makeNewAPIService("v1", 100, conditionAFalse),
			conditionType: a,
			expected:      false,
		},
		{
			name:          "Should return false when condition of type is present but status is not ConditionTrue",
			apiService:    makeNewAPIService("v1", 100, conditionATrue),
			conditionType: a,
			expected:      true,
		},
	}

	for _, testCase := range testCases {
		if isConditionTure := IsAPIServiceConditionTrue(testCase.apiService, testCase.conditionType); isConditionTure !=
			testCase.expected {
			t.Errorf("Expected condition of type %v, to be %v, actually was %v", testCase.conditionType,
				isConditionTure, testCase.expected)
		}
	}
}

func TestSetAPIServiceCondition(t *testing.T) {
	conditionA1 := makeNewAPIServiceCondition(a, "a1 reason", "a1 message", v1.ConditionTrue)
	conditionA2 := makeNewAPIServiceCondition(a, "a2 reason", "a2 message", v1.ConditionTrue)
	testCases := []*struct {
		name              string
		apiService        *v1.APIService
		conditionType     v1.APIServiceConditionType
		initialCondition  *v1.APIServiceCondition
		setCondition      v1.APIServiceCondition
		expectedCondition *v1.APIServiceCondition
	}{
		{
			name:              "Should set a new condition with type where previously there was no condition of that type",
			apiService:        makeNewAPIService("v1", 100),
			conditionType:     a,
			initialCondition:  nil,
			setCondition:      conditionA1,
			expectedCondition: &conditionA1,
		},
		{
			name:              "Should override a condition of type, when a condition of that type existed previously",
			apiService:        makeNewAPIService("v1", 100, conditionA1),
			conditionType:     a,
			initialCondition:  &conditionA1,
			setCondition:      conditionA2,
			expectedCondition: &conditionA2,
		},
	}
	for _, testCase := range testCases {
		startingCondition := GetAPIServiceConditionByType(testCase.apiService, testCase.conditionType)
		if !reflect.DeepEqual(startingCondition, testCase.initialCondition) {
			t.Errorf("Expected to find condition %s initially, actual was %s", testCase.initialCondition,
				startingCondition)
		}

		SetAPIServiceCondition(testCase.apiService, testCase.setCondition)
		actual := GetAPIServiceConditionByType(testCase.apiService, testCase.setCondition.Type)
		if !reflect.DeepEqual(actual, testCase.expectedCondition) {
			t.Errorf("Expected %s, actual %s", testCase.expectedCondition, actual)
		}
	}
}

func TestSortedByGroupAndVersion(t *testing.T) {
	testCases := []*struct {
		name     string
		versions []string
		expected []string
	}{
		{
			name:     "case1",
			versions: []string{"v1", "v2"},
			expected: []string{"v2", "v1"},
		},
		{
			name:     "case2",
			versions: []string{"v2", "v10"},
			expected: []string{"v10", "v2"},
		},
		{
			name:     "case3",
			versions: []string{"v2", "v2beta1", "v10beta2", "v10beta1", "v10alpha1", "v1"},
			expected: []string{"v2", "v1", "v10beta2", "v10beta1", "v2beta1", "v10alpha1"},
		},
		{
			name:     "case4",
			versions: []string{"v1", "v2", "test", "foo10", "final", "foo2", "foo1"},
			expected: []string{"v2", "v1", "final", "foo1", "foo10", "foo2", "test"},
		},
		{
			name:     "case5_from_documentation",
			versions: []string{"v12alpha1", "v10", "v11beta2", "v10beta3", "v3beta1", "v2", "v11alpha2", "foo1", "v1", "foo10"},
			expected: []string{"v10", "v2", "v1", "v11beta2", "v10beta3", "v3beta1", "v12alpha1", "v11alpha2", "foo1", "foo10"},
		},
	}

	for _, testCase := range testCases {
		var apiServices []*v1.APIService
		for _, version := range testCase.versions {
			apiServices = append(apiServices, makeNewAPIService(version, 100))
		}
		sortedAPIServices := SortedByGroupAndVersion(apiServices)

		var actual []string
		for _, apiService := range sortedAPIServices[0] {
			actual = append(actual, apiService.Spec.Version)
		}
		if !reflect.DeepEqual(testCase.expected, actual) {
			t.Errorf("Expected %s, actual %s", testCase.expected, actual)
		}
	}
}

func TestGetAPIServiceConditionByType(t *testing.T) {
	conditionA := makeNewAPIServiceCondition(a, "a reason", "a message", v1.ConditionTrue)
	conditionB := makeNewAPIServiceCondition(b, "b reason", "b message", v1.ConditionTrue)
	testCases := []*struct {
		name              string
		apiService        *v1.APIService
		conditionType     v1.APIServiceConditionType
		expectedCondition *v1.APIServiceCondition
	}{
		{
			name:              "Should find a matching condition from apiService",
			apiService:        makeNewAPIService("v1", 100, conditionA, conditionB),
			conditionType:     a,
			expectedCondition: &conditionA,
		},
		{
			name:              "Should not find a matching condition",
			apiService:        makeNewAPIService("v1", 100, conditionA),
			conditionType:     b,
			expectedCondition: nil,
		},
	}

	for _, testCase := range testCases {
		actual := GetAPIServiceConditionByType(testCase.apiService, testCase.conditionType)
		if !reflect.DeepEqual(testCase.expectedCondition, actual) {
			t.Errorf("Expected %s, actual %s", testCase.expectedCondition, actual)
		}
	}
}

func makeNewAPIService(version string, priority int32, conditions ...v1.APIServiceCondition) *v1.APIService {
	return &v1.APIService{Spec: v1.APIServiceSpec{Version: version, VersionPriority: priority}, Status: v1.
		APIServiceStatus{Conditions: conditions}}
}

func makeNewAPIServiceCondition(conditionType v1.APIServiceConditionType, reason, message string, status v1.ConditionStatus) v1.
	APIServiceCondition {
	return v1.APIServiceCondition{Type: conditionType, Reason: reason, Message: message, Status: status}
}

func TestAPIServiceNameToGroupVersion(t *testing.T) {
	testCases := []struct {
		name           string
		apiServiceName string
		want           schema.GroupVersion
	}{
		{
			name:           "Should name equal want name",
			apiServiceName: "v1.apiregistration.yeahfo.github.io",
			want:           schema.GroupVersion{Group: "apiregistration.yeahfo.github.io", Version: "v1"},
		},
		{
			name:           "Should name not equal want name",
			apiServiceName: "v1alphav1.k8s.io",
			want:           schema.GroupVersion{Group: "k8s.io", Version: "v1alphav1"},
		},
		{
			name:           "Should name not equal want name",
			apiServiceName: "v1.core",
			want:           schema.GroupVersion{Group: "core", Version: "v1"},
		},
	}
	for _, testCase := range testCases {
		actual := APIServiceNameToGroupVersion(testCase.apiServiceName)
		if !reflect.DeepEqual(testCase.want, actual) {
			t.Errorf("Expected %s, actual %s", testCase.want, actual)
		}
	}
}

func TestNewLocalAvailableAPIServiceCondition(t *testing.T) {
	tests := []struct {
		name string
		want v1.APIServiceCondition
	}{
		{
			name: "Should news local available apiService",
			want: NewLocalAvailableAPIServiceCondition(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want.Status != v1.ConditionTrue || tt.want.Type != v1.Available || tt.want.Reason != "Local" {
				t.Errorf("Expected %v, got %v", tt.want, tt.want)
			}
		})
	}
}
