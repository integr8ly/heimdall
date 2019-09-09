package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageMonitorSpec defines the desired state of ImageMonitor
type ImageMonitorSpec struct {
	Namespace string
	Pattern   string
}

// ImageMonitorStatus defines the observed state of ImageMonitor
type ImageMonitorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageMonitor is the Schema for the imagemonitors API
// +k8s:openapi-gen=true
type ImageMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageMonitorSpec   `json:"spec,omitempty"`
	Status ImageMonitorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageMonitorList contains a list of ImageMonitor
type ImageMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageMonitor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageMonitor{}, &ImageMonitorList{})
}
