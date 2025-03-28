package v2

type FooStatus struct {
	// The phase of a Foo is a simple, high-level summary of where the Foo is in its lifecycle
	// +optional
	Phase FooPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=FooPhase"`

	// Represents the latest available observations of a foo's current state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []FooCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,2,rep,name=conditions"`
}

// FooPhase is a label for the condition of a foo at the current time.
type FooPhase string

const (
	// FooPhaseProcessing means the pod has been accepted by the controllers, but one or more desire has not been synchorinzed
	FooPhaseProcessing FooPhase = "Processing"
	// FooPhaseReady means all conditions of foo have been meant
	FooPhaseReady FooPhase = "Ready"
)
