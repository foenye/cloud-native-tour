package apiregistration

import (
	"k8s.io/apimachinery/pkg/version"
	"sort"
)

// ByGroupPriorityMinimum implements sort.Interface
var _ sort.Interface = ByGroupPriorityMinimum{}

// ByGroupPriorityMinimum sorts with the highest group number first, then by name.
// This is not a simple reverse, because we want the name sorting to be alpha, not reverse alpha.
type ByGroupPriorityMinimum []*APIService

func (apiServices ByGroupPriorityMinimum) Len() int {
	return len(apiServices)
}
func (apiServices ByGroupPriorityMinimum) Less(i, j int) bool {
	if apiServices[i].Spec.GroupPriorityMinimum != apiServices[j].Spec.GroupPriorityMinimum {
		return apiServices[i].Spec.GroupPriorityMinimum > apiServices[j].Spec.GroupPriorityMinimum
	}
	return apiServices[i].Name < apiServices[j].Name
}
func (apiServices ByGroupPriorityMinimum) Swap(i, j int) {
	apiServices[i], apiServices[j] = apiServices[j], apiServices[i]
}

// ByVersionPriority implements sort.Interface
var _ sort.Interface = ByVersionPriority{}

// ByVersionPriority sorts with the highest version number first, then by name.
// This is not a simple reverse, because we want the name sorting to be alpha, not reverse alpha.
type ByVersionPriority []*APIService

func (apiServices ByVersionPriority) Len() int {
	return len(apiServices)
}

func (apiServices ByVersionPriority) Less(i, j int) bool {
	if apiServices[i].Spec.VersionPriority != apiServices[j].Spec.VersionPriority {
		return apiServices[i].Spec.VersionPriority > apiServices[j].Spec.VersionPriority
	}
	return version.CompareKubeAwareVersionStrings(apiServices[i].Spec.Version, apiServices[j].Spec.Version) > 0
}

func (apiServices ByVersionPriority) Swap(i, j int) {
	apiServices[i], apiServices[j] = apiServices[j], apiServices[i]
}

// SortedByGroupAndVersion sorts APIServices into their different groups, and the sorts them based on their version.
// For example, the first element of the first array contains the APIService with the highest version number, in the
// group with the highest priority; while the last element of the last array contains the APIService with the lowest
// version number, in the group with the lowest priority.
func SortedByGroupAndVersion(servers []*APIService) [][]*APIService {
	servicesByGroupPriorityMinimum := ByGroupPriorityMinimum(servers)
	sort.Sort(servicesByGroupPriorityMinimum)

	var ret [][]*APIService
	for _, curr := range servicesByGroupPriorityMinimum {
		// check to see if we already have an entry for this group
		existingIndex := -1
		for j, groupInReturn := range ret {
			if groupInReturn[0].Spec.Group == curr.Spec.Group {
				existingIndex = j
				break
			}
		}

		if existingIndex >= 0 {
			ret[existingIndex] = append(ret[existingIndex], curr)
			sort.Sort(ByVersionPriority(ret[existingIndex]))
			continue
		}
		ret = append(ret, []*APIService{curr})
	}
	return ret
}

// GetAPIServiceConditionByType gets an *APIServiceCondition by APIServiceConditionType if present.
func GetAPIServiceConditionByType(apiService *APIService, conditionType APIServiceConditionType) *APIServiceCondition {
	for idx, condition := range apiService.Status.Conditions {
		if condition.Type == conditionType {
			return &apiService.Status.Conditions[idx]
		}
	}
	return nil
}

// SetAPIServiceCondition sets the status condition. It either overwrites the existing one or creates a new one.
func SetAPIServiceCondition(apiService *APIService, newCondition APIServiceCondition) {
	existingCondition := GetAPIServiceConditionByType(apiService, newCondition.Type)
	if existingCondition == nil {
		apiService.Status.Conditions = append(apiService.Status.Conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = newCondition.LastTransitionTime
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
}

// IsAPIServiceConditionTrue indicates if the condition is present and strictly true.
func IsAPIServiceConditionTrue(apiService *APIService, conditionType APIServiceConditionType) bool {
	condition := GetAPIServiceConditionByType(apiService, conditionType)
	return condition != nil && condition.Status == ConditionTrue
}
