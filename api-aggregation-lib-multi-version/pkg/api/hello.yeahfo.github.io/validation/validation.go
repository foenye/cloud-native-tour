package validation

import (
	"github.com/yeahfo/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.yeahfo.github.io"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateFoo(foo *hello.Foo) field.ErrorList {
	allErrors := field.ErrorList{}
	allErrors = append(allErrors, validateFooSpec(&foo.Spec, field.NewPath("spec"))...)
	return allErrors
}

func validateFooSpec(fooSpec *hello.FooSpec, specPath *field.Path) field.ErrorList {
	allErrors := field.ErrorList{}

	if len(fooSpec.Image) == 0 {
		allErrors = append(allErrors, field.Required(specPath.Child("image"), ""))
	} else {
		allErrors = append(allErrors, validateFooSpecConfig(&fooSpec.Config, specPath.Child("config"))...)
	}

	return allErrors
}

func validateFooSpecConfig(fooConfig *hello.FooConfig, configPath *field.Path) field.ErrorList {
	allErrors := field.ErrorList{}

	if len(fooConfig.Msg) == 0 {
		allErrors = append(allErrors, field.Required(configPath.Child("msg"), ""))
	}

	return allErrors
}
