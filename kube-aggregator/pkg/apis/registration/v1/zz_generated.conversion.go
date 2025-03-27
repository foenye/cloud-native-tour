//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by conversion-gen. DO NOT EDIT.

package v1

import (
	unsafe "unsafe"

	registration "github.com/foenye/cloud-native-tour/kube-aggregator/pkg/apis/registration"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*APIService)(nil), (*registration.APIService)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_APIService_To_registration_APIService(a.(*APIService), b.(*registration.APIService), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registration.APIService)(nil), (*APIService)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registration_APIService_To_v1_APIService(a.(*registration.APIService), b.(*APIService), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*APIServiceCondition)(nil), (*registration.APIServiceCondition)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_APIServiceCondition_To_registration_APIServiceCondition(a.(*APIServiceCondition), b.(*registration.APIServiceCondition), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registration.APIServiceCondition)(nil), (*APIServiceCondition)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registration_APIServiceCondition_To_v1_APIServiceCondition(a.(*registration.APIServiceCondition), b.(*APIServiceCondition), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*APIServiceList)(nil), (*registration.APIServiceList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_APIServiceList_To_registration_APIServiceList(a.(*APIServiceList), b.(*registration.APIServiceList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registration.APIServiceList)(nil), (*APIServiceList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registration_APIServiceList_To_v1_APIServiceList(a.(*registration.APIServiceList), b.(*APIServiceList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*APIServiceSpec)(nil), (*registration.APIServiceSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_APIServiceSpec_To_registration_APIServiceSpec(a.(*APIServiceSpec), b.(*registration.APIServiceSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registration.APIServiceSpec)(nil), (*APIServiceSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registration_APIServiceSpec_To_v1_APIServiceSpec(a.(*registration.APIServiceSpec), b.(*APIServiceSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*APIServiceStatus)(nil), (*registration.APIServiceStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_APIServiceStatus_To_registration_APIServiceStatus(a.(*APIServiceStatus), b.(*registration.APIServiceStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registration.APIServiceStatus)(nil), (*APIServiceStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registration_APIServiceStatus_To_v1_APIServiceStatus(a.(*registration.APIServiceStatus), b.(*APIServiceStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ServiceReference)(nil), (*registration.ServiceReference)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_ServiceReference_To_registration_ServiceReference(a.(*ServiceReference), b.(*registration.ServiceReference), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registration.ServiceReference)(nil), (*ServiceReference)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registration_ServiceReference_To_v1_ServiceReference(a.(*registration.ServiceReference), b.(*ServiceReference), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1_APIService_To_registration_APIService(in *APIService, out *registration.APIService, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1_APIServiceSpec_To_registration_APIServiceSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1_APIServiceStatus_To_registration_APIServiceStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1_APIService_To_registration_APIService is an autogenerated conversion function.
func Convert_v1_APIService_To_registration_APIService(in *APIService, out *registration.APIService, s conversion.Scope) error {
	return autoConvert_v1_APIService_To_registration_APIService(in, out, s)
}

func autoConvert_registration_APIService_To_v1_APIService(in *registration.APIService, out *APIService, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_registration_APIServiceSpec_To_v1_APIServiceSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_registration_APIServiceStatus_To_v1_APIServiceStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_registration_APIService_To_v1_APIService is an autogenerated conversion function.
func Convert_registration_APIService_To_v1_APIService(in *registration.APIService, out *APIService, s conversion.Scope) error {
	return autoConvert_registration_APIService_To_v1_APIService(in, out, s)
}

func autoConvert_v1_APIServiceCondition_To_registration_APIServiceCondition(in *APIServiceCondition, out *registration.APIServiceCondition, s conversion.Scope) error {
	out.Type = registration.APIServiceConditionType(in.Type)
	out.Status = registration.ConditionStatus(in.Status)
	out.LastTransitionTime = in.LastTransitionTime
	out.Reason = in.Reason
	out.Message = in.Message
	return nil
}

// Convert_v1_APIServiceCondition_To_registration_APIServiceCondition is an autogenerated conversion function.
func Convert_v1_APIServiceCondition_To_registration_APIServiceCondition(in *APIServiceCondition, out *registration.APIServiceCondition, s conversion.Scope) error {
	return autoConvert_v1_APIServiceCondition_To_registration_APIServiceCondition(in, out, s)
}

func autoConvert_registration_APIServiceCondition_To_v1_APIServiceCondition(in *registration.APIServiceCondition, out *APIServiceCondition, s conversion.Scope) error {
	out.Type = APIServiceConditionType(in.Type)
	out.Status = ConditionStatus(in.Status)
	out.LastTransitionTime = in.LastTransitionTime
	out.Reason = in.Reason
	out.Message = in.Message
	return nil
}

// Convert_registration_APIServiceCondition_To_v1_APIServiceCondition is an autogenerated conversion function.
func Convert_registration_APIServiceCondition_To_v1_APIServiceCondition(in *registration.APIServiceCondition, out *APIServiceCondition, s conversion.Scope) error {
	return autoConvert_registration_APIServiceCondition_To_v1_APIServiceCondition(in, out, s)
}

func autoConvert_v1_APIServiceList_To_registration_APIServiceList(in *APIServiceList, out *registration.APIServiceList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]registration.APIService, len(*in))
		for i := range *in {
			if err := Convert_v1_APIService_To_registration_APIService(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1_APIServiceList_To_registration_APIServiceList is an autogenerated conversion function.
func Convert_v1_APIServiceList_To_registration_APIServiceList(in *APIServiceList, out *registration.APIServiceList, s conversion.Scope) error {
	return autoConvert_v1_APIServiceList_To_registration_APIServiceList(in, out, s)
}

func autoConvert_registration_APIServiceList_To_v1_APIServiceList(in *registration.APIServiceList, out *APIServiceList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]APIService, len(*in))
		for i := range *in {
			if err := Convert_registration_APIService_To_v1_APIService(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_registration_APIServiceList_To_v1_APIServiceList is an autogenerated conversion function.
func Convert_registration_APIServiceList_To_v1_APIServiceList(in *registration.APIServiceList, out *APIServiceList, s conversion.Scope) error {
	return autoConvert_registration_APIServiceList_To_v1_APIServiceList(in, out, s)
}

func autoConvert_v1_APIServiceSpec_To_registration_APIServiceSpec(in *APIServiceSpec, out *registration.APIServiceSpec, s conversion.Scope) error {
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(registration.ServiceReference)
		if err := Convert_v1_ServiceReference_To_registration_ServiceReference(*in, *out, s); err != nil {
			return err
		}
	} else {
		out.Service = nil
	}
	out.Group = in.Group
	out.Version = in.Version
	out.InsecureSkipTLSVerify = in.InsecureSkipTLSVerify
	out.CABundle = *(*[]byte)(unsafe.Pointer(&in.CABundle))
	out.GroupPriorityMinimum = in.GroupPriorityMinimum
	out.VersionPriority = in.VersionPriority
	return nil
}

// Convert_v1_APIServiceSpec_To_registration_APIServiceSpec is an autogenerated conversion function.
func Convert_v1_APIServiceSpec_To_registration_APIServiceSpec(in *APIServiceSpec, out *registration.APIServiceSpec, s conversion.Scope) error {
	return autoConvert_v1_APIServiceSpec_To_registration_APIServiceSpec(in, out, s)
}

func autoConvert_registration_APIServiceSpec_To_v1_APIServiceSpec(in *registration.APIServiceSpec, out *APIServiceSpec, s conversion.Scope) error {
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceReference)
		if err := Convert_registration_ServiceReference_To_v1_ServiceReference(*in, *out, s); err != nil {
			return err
		}
	} else {
		out.Service = nil
	}
	out.Group = in.Group
	out.Version = in.Version
	out.InsecureSkipTLSVerify = in.InsecureSkipTLSVerify
	out.CABundle = *(*[]byte)(unsafe.Pointer(&in.CABundle))
	out.GroupPriorityMinimum = in.GroupPriorityMinimum
	out.VersionPriority = in.VersionPriority
	return nil
}

// Convert_registration_APIServiceSpec_To_v1_APIServiceSpec is an autogenerated conversion function.
func Convert_registration_APIServiceSpec_To_v1_APIServiceSpec(in *registration.APIServiceSpec, out *APIServiceSpec, s conversion.Scope) error {
	return autoConvert_registration_APIServiceSpec_To_v1_APIServiceSpec(in, out, s)
}

func autoConvert_v1_APIServiceStatus_To_registration_APIServiceStatus(in *APIServiceStatus, out *registration.APIServiceStatus, s conversion.Scope) error {
	out.Conditions = *(*[]registration.APIServiceCondition)(unsafe.Pointer(&in.Conditions))
	return nil
}

// Convert_v1_APIServiceStatus_To_registration_APIServiceStatus is an autogenerated conversion function.
func Convert_v1_APIServiceStatus_To_registration_APIServiceStatus(in *APIServiceStatus, out *registration.APIServiceStatus, s conversion.Scope) error {
	return autoConvert_v1_APIServiceStatus_To_registration_APIServiceStatus(in, out, s)
}

func autoConvert_registration_APIServiceStatus_To_v1_APIServiceStatus(in *registration.APIServiceStatus, out *APIServiceStatus, s conversion.Scope) error {
	out.Conditions = *(*[]APIServiceCondition)(unsafe.Pointer(&in.Conditions))
	return nil
}

// Convert_registration_APIServiceStatus_To_v1_APIServiceStatus is an autogenerated conversion function.
func Convert_registration_APIServiceStatus_To_v1_APIServiceStatus(in *registration.APIServiceStatus, out *APIServiceStatus, s conversion.Scope) error {
	return autoConvert_registration_APIServiceStatus_To_v1_APIServiceStatus(in, out, s)
}

func autoConvert_v1_ServiceReference_To_registration_ServiceReference(in *ServiceReference, out *registration.ServiceReference, s conversion.Scope) error {
	out.Namespace = in.Namespace
	out.Name = in.Name
	if err := metav1.Convert_Pointer_int32_To_int32(&in.Port, &out.Port, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1_ServiceReference_To_registration_ServiceReference is an autogenerated conversion function.
func Convert_v1_ServiceReference_To_registration_ServiceReference(in *ServiceReference, out *registration.ServiceReference, s conversion.Scope) error {
	return autoConvert_v1_ServiceReference_To_registration_ServiceReference(in, out, s)
}

func autoConvert_registration_ServiceReference_To_v1_ServiceReference(in *registration.ServiceReference, out *ServiceReference, s conversion.Scope) error {
	out.Namespace = in.Namespace
	out.Name = in.Name
	if err := metav1.Convert_int32_To_Pointer_int32(&in.Port, &out.Port, s); err != nil {
		return err
	}
	return nil
}

// Convert_registration_ServiceReference_To_v1_ServiceReference is an autogenerated conversion function.
func Convert_registration_ServiceReference_To_v1_ServiceReference(in *registration.ServiceReference, out *ServiceReference, s conversion.Scope) error {
	return autoConvert_registration_ServiceReference_To_v1_ServiceReference(in, out, s)
}
