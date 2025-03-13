package aggregator

import (
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"sort"
)

var _ sort.Interface = byPriority{}

// byPriority can be used in sort.Sort to sort specs with their priorities.
type byPriority struct {
	apiServices     []*apiregistrationv1.APIService
	groupPriorities map[string]int32
}

func (sorter byPriority) Len() int {
	return len(sorter.apiServices)
}

func (sorter byPriority) Less(i, j int) bool {
	// All local specs will come first
	if sorter.apiServices[i].Spec.Service == nil && sorter.apiServices[j].Spec.Service != nil {
		return true
	}
	if sorter.apiServices[i].Spec.Service != nil && sorter.apiServices[j].Spec.Service == nil {
		return false
	}
	// WARNING: This will result in not following priorities for local APIService.
	if sorter.apiServices[i].Spec.Service == nil {
		return sorter.apiServices[i].Name < sorter.apiServices[j].Name
	}
	var iPriority, jPriority int32
	if sorter.apiServices[i].Spec.Group == sorter.apiServices[j].Spec.Group {
		iPriority = sorter.apiServices[i].Spec.VersionPriority
		jPriority = sorter.apiServices[i].Spec.VersionPriority
	} else {
		iPriority = sorter.groupPriorities[sorter.apiServices[i].Spec.Group]
		jPriority = sorter.groupPriorities[sorter.apiServices[j].Spec.Group]
	}
	if iPriority != jPriority {
		return iPriority > jPriority
	}
	// Sort by service name.
	return sorter.apiServices[i].Name < sorter.apiServices[j].Name
}

func (sorter byPriority) Swap(i, j int) {
	sorter.apiServices[i], sorter.apiServices[j] = sorter.apiServices[j], sorter.apiServices[i]
}

func sortByPriority(apiServices []*apiregistrationv1.APIService) {
	sorter := byPriority{
		apiServices:     apiServices,
		groupPriorities: map[string]int32{},
	}

	for _, apiService := range apiServices {
		if apiService.Spec.Service == nil {
			continue
		}
		if groupPriorityMinimum, found := sorter.groupPriorities[apiService.Spec.Group]; !found || apiService.Spec.
			GroupPriorityMinimum > groupPriorityMinimum {
			sorter.groupPriorities[apiService.Spec.Group] = apiService.Spec.GroupPriorityMinimum
		}
	}
	sort.Sort(sorter)
}
