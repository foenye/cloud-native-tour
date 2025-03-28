// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: github.com/foenye/cloud-native-tour/crd-getting-started/pkg/apis/greeting/v1/generated.proto

package v1

import (
	fmt "fmt"

	io "io"

	proto "github.com/gogo/protobuf/proto"

	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

func (m *Foo) Reset()      { *m = Foo{} }
func (*Foo) ProtoMessage() {}
func (*Foo) Descriptor() ([]byte, []int) {
	return fileDescriptor_d48eb841b48a883b, []int{0}
}
func (m *Foo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Foo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Foo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Foo.Merge(m, src)
}
func (m *Foo) XXX_Size() int {
	return m.Size()
}
func (m *Foo) XXX_DiscardUnknown() {
	xxx_messageInfo_Foo.DiscardUnknown(m)
}

var xxx_messageInfo_Foo proto.InternalMessageInfo

func (m *FooList) Reset()      { *m = FooList{} }
func (*FooList) ProtoMessage() {}
func (*FooList) Descriptor() ([]byte, []int) {
	return fileDescriptor_d48eb841b48a883b, []int{1}
}
func (m *FooList) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *FooList) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *FooList) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FooList.Merge(m, src)
}
func (m *FooList) XXX_Size() int {
	return m.Size()
}
func (m *FooList) XXX_DiscardUnknown() {
	xxx_messageInfo_FooList.DiscardUnknown(m)
}

var xxx_messageInfo_FooList proto.InternalMessageInfo

func (m *FooSpec) Reset()      { *m = FooSpec{} }
func (*FooSpec) ProtoMessage() {}
func (*FooSpec) Descriptor() ([]byte, []int) {
	return fileDescriptor_d48eb841b48a883b, []int{2}
}
func (m *FooSpec) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *FooSpec) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *FooSpec) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FooSpec.Merge(m, src)
}
func (m *FooSpec) XXX_Size() int {
	return m.Size()
}
func (m *FooSpec) XXX_DiscardUnknown() {
	xxx_messageInfo_FooSpec.DiscardUnknown(m)
}

var xxx_messageInfo_FooSpec proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Foo)(nil), "github.com.foenye.cloud_native_tour.crd_getting_started.pkg.apis.greeting.v1.Foo")
	proto.RegisterType((*FooList)(nil), "github.com.foenye.cloud_native_tour.crd_getting_started.pkg.apis.greeting.v1.FooList")
	proto.RegisterType((*FooSpec)(nil), "github.com.foenye.cloud_native_tour.crd_getting_started.pkg.apis.greeting.v1.FooSpec")
}

func init() {
	proto.RegisterFile("github.com/foenye/cloud-native-tour/crd-getting-started/pkg/apis/greeting/v1/generated.proto", fileDescriptor_d48eb841b48a883b)
}

var fileDescriptor_d48eb841b48a883b = []byte{
	// 470 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x93, 0x4f, 0x6f, 0xd3, 0x30,
	0x18, 0xc6, 0x9b, 0xfd, 0x51, 0xb7, 0x14, 0x04, 0x0a, 0x97, 0xaa, 0x07, 0x6f, 0xea, 0x69, 0x1c,
	0x62, 0xd3, 0x09, 0x10, 0xe7, 0x08, 0x55, 0x42, 0xea, 0x84, 0x08, 0xe2, 0x82, 0x26, 0x15, 0xd7,
	0x79, 0xe7, 0x9a, 0x92, 0x38, 0xb2, 0x9d, 0xa0, 0xdd, 0xf8, 0x08, 0x7c, 0xac, 0x1e, 0x77, 0xdc,
	0x69, 0xac, 0xe1, 0x8b, 0x20, 0x3b, 0x29, 0xa9, 0x18, 0x88, 0x1d, 0x7a, 0xab, 0xfd, 0xbe, 0xcf,
	0xef, 0x79, 0x9e, 0x5a, 0xf1, 0xcf, 0xb9, 0x30, 0xf3, 0x62, 0x86, 0x99, 0x4c, 0xc9, 0x85, 0x84,
	0xec, 0x12, 0x08, 0xfb, 0x22, 0x8b, 0x24, 0xcc, 0xa8, 0x11, 0x25, 0x84, 0x46, 0x16, 0x8a, 0x30,
	0x95, 0x84, 0x1c, 0x8c, 0x11, 0x19, 0x0f, 0xb5, 0xa1, 0xca, 0x40, 0x42, 0xf2, 0x05, 0x27, 0x34,
	0x17, 0x9a, 0x70, 0x05, 0x60, 0x27, 0xa4, 0x1c, 0x11, 0x0e, 0x19, 0x28, 0x6a, 0x20, 0xc1, 0xb9,
	0x92, 0x46, 0x06, 0x93, 0x96, 0x8e, 0x6b, 0x3a, 0x76, 0xf4, 0x69, 0x4d, 0x9f, 0x5a, 0x3a, 0x66,
	0x2a, 0x99, 0x36, 0xf4, 0x69, 0x43, 0xc7, 0xf9, 0x82, 0x63, 0x4b, 0xc7, 0x6b, 0x3a, 0x2e, 0x47,
	0x83, 0x70, 0x23, 0x2b, 0x97, 0x5c, 0x12, 0x67, 0x32, 0x2b, 0x2e, 0xdc, 0xc9, 0x1d, 0xdc, 0xaf,
	0xda, 0x7c, 0xf0, 0x7c, 0xf1, 0x4a, 0x63, 0x21, 0x6d, 0xca, 0x94, 0xb2, 0xb9, 0xc8, 0x40, 0x5d,
	0xb6, 0xb1, 0x53, 0x30, 0xf4, 0x2f, 0x91, 0x07, 0xe4, 0x5f, 0x2a, 0x55, 0x64, 0x46, 0xa4, 0x70,
	0x47, 0xf0, 0xf2, 0x7f, 0x02, 0xcd, 0xe6, 0x90, 0xd2, 0x3f, 0x75, 0xc3, 0x5b, 0xcf, 0xdf, 0x1d,
	0x4b, 0x19, 0x7c, 0xf2, 0x0f, 0x6c, 0x96, 0x84, 0x1a, 0xda, 0xf7, 0x8e, 0xbd, 0x93, 0xde, 0xe9,
	0x33, 0x5c, 0x23, 0xf1, 0x26, 0xb2, 0xfd, 0x4b, 0xec, 0x36, 0x2e, 0x47, 0xf8, 0xed, 0xec, 0x33,
	0x30, 0x73, 0x06, 0x86, 0x46, 0xc1, 0xf2, 0xe6, 0xa8, 0x53, 0xdd, 0x1c, 0xf9, 0xed, 0x5d, 0xfc,
	0x9b, 0x1a, 0x7c, 0xf5, 0xf7, 0x74, 0x0e, 0xac, 0xbf, 0xe3, 0xe8, 0x1f, 0xf0, 0x36, 0x1f, 0x05,
	0x8f, 0xa5, 0x7c, 0x9f, 0x03, 0x8b, 0x1e, 0x34, 0x11, 0xf6, 0xec, 0x29, 0x76, 0x86, 0xc3, 0x1f,
	0x9e, 0xdf, 0x1d, 0x4b, 0x39, 0x11, 0xda, 0x04, 0xe7, 0x77, 0x6a, 0xe2, 0xfb, 0xd5, 0xb4, 0x6a,
	0x57, 0xf2, 0x71, 0xe3, 0x70, 0xb0, 0xbe, 0xd9, 0xa8, 0x58, 0xfa, 0xfb, 0xc2, 0x40, 0xaa, 0xfb,
	0x3b, 0xc7, 0xbb, 0x27, 0xbd, 0xd3, 0x77, 0x5b, 0xef, 0x18, 0x3d, 0x6c, 0xdc, 0xf7, 0xdf, 0x58,
	0x9f, 0xb8, 0xb6, 0x1b, 0x2e, 0x5c, 0x41, 0x5b, 0x39, 0x78, 0xea, 0x77, 0x53, 0xd0, 0x9a, 0x72,
	0x70, 0xfd, 0x0e, 0xa3, 0x47, 0x8d, 0xa2, 0x7b, 0x56, 0x5f, 0xc7, 0xeb, 0x79, 0xf0, 0xc2, 0xef,
	0x25, 0xa0, 0x99, 0x12, 0xb9, 0x11, 0x32, 0x73, 0xef, 0x72, 0x18, 0x3d, 0x69, 0xd6, 0x7b, 0xaf,
	0xdb, 0x51, 0xbc, 0xb9, 0x17, 0xa9, 0xe5, 0x0a, 0x75, 0xae, 0x56, 0xa8, 0x73, 0xbd, 0x42, 0x9d,
	0x6f, 0x15, 0xf2, 0x96, 0x15, 0xf2, 0xae, 0x2a, 0xe4, 0x5d, 0x57, 0xc8, 0xbb, 0xad, 0x90, 0xf7,
	0xfd, 0x27, 0xea, 0x7c, 0x9c, 0x6c, 0xf3, 0x8b, 0xfe, 0x15, 0x00, 0x00, 0xff, 0xff, 0x64, 0x97,
	0x96, 0xb2, 0x20, 0x04, 0x00, 0x00,
}

func (m *Foo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Foo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Foo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Spec.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenerated(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size, err := m.ObjectMeta.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenerated(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *FooList) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *FooList) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *FooList) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Items) > 0 {
		for iNdEx := len(m.Items) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Items[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenerated(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	{
		size, err := m.ListMeta.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenerated(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *FooSpec) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *FooSpec) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *FooSpec) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	i -= len(m.Description)
	copy(dAtA[i:], m.Description)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Description)))
	i--
	dAtA[i] = 0x12
	i -= len(m.Message)
	copy(dAtA[i:], m.Message)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Message)))
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintGenerated(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenerated(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Foo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.ObjectMeta.Size()
	n += 1 + l + sovGenerated(uint64(l))
	l = m.Spec.Size()
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func (m *FooList) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.ListMeta.Size()
	n += 1 + l + sovGenerated(uint64(l))
	if len(m.Items) > 0 {
		for _, e := range m.Items {
			l = e.Size()
			n += 1 + l + sovGenerated(uint64(l))
		}
	}
	return n
}

func (m *FooSpec) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Message)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Description)
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func sovGenerated(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenerated(x uint64) (n int) {
	return sovGenerated(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Foo) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Foo{`,
		`ObjectMeta:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.ObjectMeta), "ObjectMeta", "v1.ObjectMeta", 1), `&`, ``, 1) + `,`,
		`Spec:` + strings.Replace(strings.Replace(this.Spec.String(), "FooSpec", "FooSpec", 1), `&`, ``, 1) + `,`,
		`}`,
	}, "")
	return s
}
func (this *FooList) String() string {
	if this == nil {
		return "nil"
	}
	repeatedStringForItems := "[]Foo{"
	for _, f := range this.Items {
		repeatedStringForItems += strings.Replace(strings.Replace(f.String(), "Foo", "Foo", 1), `&`, ``, 1) + ","
	}
	repeatedStringForItems += "}"
	s := strings.Join([]string{`&FooList{`,
		`ListMeta:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.ListMeta), "ListMeta", "v1.ListMeta", 1), `&`, ``, 1) + `,`,
		`Items:` + repeatedStringForItems + `,`,
		`}`,
	}, "")
	return s
}
func (this *FooSpec) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&FooSpec{`,
		`Message:` + fmt.Sprintf("%v", this.Message) + `,`,
		`Description:` + fmt.Sprintf("%v", this.Description) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringGenerated(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Foo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Foo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Foo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ObjectMeta", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ObjectMeta.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Spec", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Spec.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *FooList) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: FooList: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: FooList: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ListMeta", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ListMeta.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Items", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Items = append(m.Items, Foo{})
			if err := m.Items[len(m.Items)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *FooSpec) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: FooSpec: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: FooSpec: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Message", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Message = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipGenerated(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthGenerated
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenerated
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenerated
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenerated        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenerated          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenerated = fmt.Errorf("proto: unexpected end of group")
)
