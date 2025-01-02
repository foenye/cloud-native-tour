package validation

import (
	"fmt"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/api/validation/path"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"strings"
)

// ValidateAPIService validates that the APIService is correctly defined.
func ValidateAPIService(apiService *apiregistration.APIService) field.ErrorList {
	requiredName := strings.Join([]string{apiService.Spec.Version, apiService.Spec.Group}, ".")

	allErrs := validation.ValidateObjectMeta(&apiService.ObjectMeta, false,
		func(name string, prefix bool) []string {
			if minimalFailures := path.IsValidPathSegmentName(name); len(minimalFailures) > 0 {
				return minimalFailures
			}
			// the name *must* be version.group
			if name != requiredName {
				return []string{fmt.Sprintf("must be `spec.version +\".\"+spec.group`: %q", requiredName)}
			}
			return []string{}
		}, field.NewPath("metadata"))

	// in this case we allow empty group
	if len(apiService.Spec.Group) == 0 && apiService.Spec.Version != "v1" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "group"),
			"only v1 may have an empty group and it better be legacy kube"))
	}

	if len(apiService.Spec.Group) > 0 {
		for _, dns1123ErrString := range utilvalidation.IsDNS1123Subdomain(apiService.Spec.Group) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "group"),
				apiService.Spec.Group, dns1123ErrString))
		}
	}

	for _, dns1035ErrString := range utilvalidation.IsDNS1035Label(apiService.Spec.Version) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "version"),
			apiService.Spec.Version, dns1035ErrString))
	}

	if apiService.Spec.GroupPriorityMinimum <= 0 || apiService.Spec.GroupPriorityMinimum > 2000 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "groupPriorityMinimum"),
			apiService.Spec.GroupPriorityMinimum, "must be positive and less than 2000"))
	}

	if apiService.Spec.VersionPriority <= 0 || apiService.Spec.VersionPriority > 1000 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "versionPriority"),
			apiService.Spec.VersionPriority, "must be positive and less than 1000"))
	}

	if apiService.Spec.Service == nil {
		if len(apiService.Spec.CABundle) != 0 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "caBundle"),
				fmt.Sprintf("%d bytes", len(apiService.Spec.CABundle)), "local APIServices may not hava a caBundle"))
		}
		if apiService.Spec.InsecureSkipTLSVerify {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "insecureSkipTLSVerify"),
				apiService.Spec.InsecureSkipTLSVerify, "local APIServices may not hava insecureSkipTLSVerify"))
		}
		return allErrs
	}

	if len(apiService.Spec.Service.Namespace) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "service", "namespace"),
			""))
	}
	if len(apiService.Spec.Service.Name) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "service", "name"),
			""))
	}
	if errs := utilvalidation.IsValidPortNum(int(apiService.Spec.Service.Port)); errs != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "service", "port"),
			apiService.Spec.Service.Port, "port is not valid"+strings.Join(errs, ", ")))
	}
	if apiService.Spec.InsecureSkipTLSVerify && len(apiService.Spec.CABundle) > 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "insecureSkipTLSVerify"),
			apiService.Spec.Service.Port, "may not be true if caBundle is present"))
	}

	return allErrs
}

// ValidateAPIServiceUpdate validates an update of APIService.
func ValidateAPIServiceUpdate(newAPIService *apiregistration.APIService, oldAPIService *apiregistration.
	APIService) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&newAPIService.ObjectMeta, &oldAPIService.ObjectMeta, field.NewPath(
		"metadata"))
	allErrs = append(allErrs, ValidateAPIService(newAPIService)...)
	return allErrs
}

// ValidateAPIServiceStatus validates that the APIService status is one of 'True', 'False' or 'Unknown'.
func ValidateAPIServiceStatus(status *apiregistration.APIServiceStatus, fieldPath *field.Path) (allErrs field.ErrorList) {
	for i, condition := range status.Conditions {
		if condition.Status != apiregistration.ConditionTrue &&
			condition.Status != apiregistration.ConditionFalse &&
			condition.Status != apiregistration.ConditionUnknown {
			allErrs = append(allErrs, field.NotSupported(fieldPath.Child("conditions").Index(i).Child(
				"status"), condition.Status, []string{
				string(apiregistration.ConditionTrue),
				string(apiregistration.ConditionFalse),
				string(apiregistration.ConditionUnknown),
			}))
		}
	}
	return allErrs
}

// ValidateAPIServiceStatusUpdate validates an update of the status field of APIService.
func ValidateAPIServiceStatusUpdate(newAPIService *apiregistration.APIService, oldAPIService *apiregistration.
	APIService) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&newAPIService.ObjectMeta, &oldAPIService.ObjectMeta, field.NewPath(
		"metadata"))
	allErrs = append(allErrs, ValidateAPIServiceStatus(&newAPIService.Status, field.NewPath("status"))...)
	return allErrs
}
