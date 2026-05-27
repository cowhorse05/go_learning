package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// SimpleApp is our custom resource
type SimpleApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SimpleAppSpec   `json:"spec,omitempty"`
	Status            SimpleAppStatus `json:"status,omitempty"`
}

type SimpleAppSpec struct {
	Image    string `json:"image"`
	Replicas int32  `json:"replicas"`
	Port     int32  `json:"port"`
}

type SimpleAppStatus struct {
	ReadyReplicas int32  `json:"readyReplicas"`
	Phase         string `json:"phase"`
}

// SimpleAppList contains a list of SimpleApp
type SimpleAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SimpleApp `json:"items"`
}
