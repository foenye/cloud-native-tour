package v2

type FooConfig struct {
	// Message says hello world!
	Message string `json:"message" protobuf:"bytes,1,opt,name=message"`
	// Description provides some verbose information
	// +optional
	Description string `json:"description,omitempty" protobuf:"bytes,2,opt,name=description"`
}
