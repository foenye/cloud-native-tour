package apiregistration

import (
	"reflect"
	"testing"
)

var (
	a APIServiceConditionType = "A"
	b APIServiceConditionType = "B"
)

func TestGetAPIServiceConditionByType(t *testing.T) {
	conditionA := makeNewAPIServiceCondition(a, "a reason", "a message", ConditionTrue)
	conditionB := makeNewAPIServiceCondition(b, "b reason", "b message", ConditionTrue)
	tests := []*struct {
		name              string
		apiService        *APIService
		conditionType     APIServiceConditionType
		expectedCondition *APIServiceCondition
	}{
		{
			name:              "Should find a matching condition for apiService",
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
	for _, testCase := range tests {
		actual := GetAPIServiceConditionByType(testCase.apiService, testCase.conditionType)
		if !reflect.DeepEqual(testCase.expectedCondition, actual) {
			t.Errorf("expected %s, actual %s", testCase.expectedCondition, actual)
		}
	}
}

func TestIsAPIServiceConditionTrue(t *testing.T) {
	conditionATrue := makeNewAPIServiceCondition(a, "a reason", "a message", ConditionTrue)
	conditionAFalse := makeNewAPIServiceCondition(a, "a reason", "a message", ConditionFalse)
	tests := []*struct {
		name          string
		apiService    *APIService
		conditionType APIServiceConditionType
		expected      bool
	}{
		{
			name:          "Should return false when condition of type is not present",
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

	for _, testCase := range tests {
		if isConditionTrue := IsAPIServiceConditionTrue(testCase.apiService, testCase.conditionType); isConditionTrue != testCase.expected {
			t.Errorf("expected conditon of type %v be to %v actully was %v",
				testCase.conditionType, isConditionTrue, testCase.expected)
		}
	}
}

func TestSetAPIServiceCondition(t *testing.T) {
	conditionA1 := makeNewAPIServiceCondition(a, "a1 reason", "a1 message", ConditionTrue)
	conditionA2 := makeNewAPIServiceCondition(a, "a2 reason", "a2 message", ConditionTrue)
	tests := []*struct {
		name              string
		apiService        *APIService
		conditionType     APIServiceConditionType
		initialCondition  *APIServiceCondition
		setCondition      APIServiceCondition
		expectedCondition *APIServiceCondition
	}{
		{
			name:              "Should set a new condition with where previously there was not condition of that type",
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

	for _, testCase := range tests {
		startingCondition := GetAPIServiceConditionByType(testCase.apiService, testCase.conditionType)
		if !reflect.DeepEqual(startingCondition, testCase.initialCondition) {
			t.Errorf("expected to find condition %s initially, actual was %s", testCase.initialCondition,
				startingCondition)
		}
		SetAPIServiceCondition(testCase.apiService, testCase.setCondition)
		actual := GetAPIServiceConditionByType(testCase.apiService, testCase.setCondition.Type)
		if !reflect.DeepEqual(actual, testCase.expectedCondition) {
			t.Errorf("expected %s, actual %s", testCase.expectedCondition, actual)
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
			name:     "Case1",
			versions: []string{"v1", "v2"},
			expected: []string{"v2", "v1"},
		},
		{
			name:     "Case2",
			versions: []string{"v2", "v10"},
			expected: []string{"v10", "v2"},
		},
		{
			name:     "Case3",
			versions: []string{"v2", "v2beta1", "v10beta2", "v10beta1", "v10alpha1", "v1"},
			expected: []string{"v2", "v1", "v10beta2", "v10beta1", "v2beta1", "v10alpha1"},
		},
		{
			name:     "Case4",
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
		apiServices := make([]*APIService, 0)
		for _, version := range testCase.versions {
			apiServices = append(apiServices, makeNewAPIService(version, 100))
		}
		sortedAPIServices := SortedByGroupAndVersion(apiServices)
		actual := make([]string, 0)
		for _, apiService := range sortedAPIServices[0] {
			actual = append(actual, apiService.Spec.Version)
		}
		if !reflect.DeepEqual(testCase.expected, actual) {
			t.Errorf("expected %s, actual %s", testCase.expected, actual)
		}
	}
}

func makeNewAPIService(version string, priority int32, conditions ...APIServiceCondition) *APIService {
	status := APIServiceStatus{Conditions: conditions}
	return &APIService{Spec: APIServiceSpec{Version: version, VersionPriority: priority}, Status: status}
}

func makeNewAPIServiceCondition(conditionType APIServiceConditionType, reason, message string, status ConditionStatus) APIServiceCondition {
	return APIServiceCondition{Type: conditionType, Reason: reason, Message: message, Status: status}
}
