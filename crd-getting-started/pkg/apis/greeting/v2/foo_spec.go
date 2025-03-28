package v2

type FooSpec struct {
	// Container image that the container is running to do our foo work
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// Config is the configuration used by foo container
	Config FooConfig `json:"config" protobuf:"bytes,2,opt,name=config"`
}
