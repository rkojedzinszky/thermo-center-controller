/*
MIT License

Copyright (c) 2020 Richard Kojedzinszky

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalMemcached represents an external memcached instance to be used
type ExternalMemcached struct {
	// Hostname of memcached
	Hostname string `json:"hostname"`

	// Port of memcached, defaults to 11211
	Port int `json:"port,omitempty"`
}

// ExternalMQTT represents an external MQTT instance to be used
type ExternalMQTT struct {
	// Hostname of MQTT broker
	Hostname string `json:"hostname"`

	// Port of MQTT broker, defaults to 1883
	Port int `json:"port,omitempty"`
}

// Database specifies database connection parameters
type Database struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// Deployment base parameters
type Deployment struct {
	Image    string `json:"image,omitempty"`
	Replicas *int32 `json:"replicas,omitempty"`
}

// Ingress parameters
type Ingress struct {
	// HostNames is a list of DNS domain names which will point
	// to a ThermoCenter installation
	// +listType=atomic
	HostNames []string `json:"hostNames"`

	// TLS specifies whether to generate tls section in Kubernetes Ingress resource
	TLS bool `json:"tls,omitempty"`

	// Extra annotations to add to Kubernetes Ingress resource
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Graphite parameters
type Graphite struct {
	// Hostname specifies the hostname for Graphite/Carbon line receiver
	Hostname string `json:"host"`

	// Port Specifies the port for Graphite/Carbon line receiver
	Port uint16 `json:"port,omitempty"`

	// MetricPathTemplate is used to construct Graphite metric paths.
	// During interpolation, `SensorID` and `Metric` fields are available.
	MetricPathTemplate string `json:"metricPathTemplate,omitempty"`
}

// ThermoCenterSpec defines the desired state of ThermoCenter
type ThermoCenterSpec struct {
	// Ingress represents Ingress parameters
	Ingress Ingress `json:"ingress"`

	// ExternalMemcached points to an external memcached instance
	ExternalMemcached *ExternalMemcached `json:"externalMemcached,omitempty"`

	// ExternalMQTT points to an external mqtt instance
	ExternalMQTT *ExternalMQTT `json:"externalMQTT,omitempty"`

	// Version defines the desired version. If empty, uses 'latest' tag for all images
	Version *string `json:"version,omitempty"`

	// Desired replicas of all components, defaults to 1
	Replicas int32 `json:"replicas"`

	// Postgresql access configuration
	Database *Database `json:"database,omitempty"`

	// Deployment specifications, on production deployments these are typically not specified
	UI       *Deployment `json:"ui,omitempty"`
	API      *Deployment `json:"api,omitempty"`
	WS       *Deployment `json:"ws,omitempty"`
	GRPC     *Deployment `json:"grpc,omitempty"`
	Receiver *Deployment `json:"receiver,omitempty"`

	// Graphite parameter specification
	Graphite Graphite `json:"graphite,omitempty"`
}

// ThermoCenterStatus defines the observed state of ThermoCenter
type ThermoCenterStatus struct {
	DatabaseVersion string `json:"databaseVersion"`
	Status          string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=.status.databaseVersion,description="Database version",name=DBVer,type=string
// +kubebuilder:printcolumn:JSONPath=.status.status,description="ThermoCenter status",name=Status,type=string

// ThermoCenter is the Schema for the thermocenters API
type ThermoCenter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ThermoCenterSpec   `json:"spec,omitempty"`
	Status ThermoCenterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ThermoCenterList contains a list of ThermoCenter
type ThermoCenterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ThermoCenter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ThermoCenter{}, &ThermoCenterList{})
}
