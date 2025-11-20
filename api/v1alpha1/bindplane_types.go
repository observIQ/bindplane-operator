/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BindplaneSpec defines the desired state of Bindplane.
type BindplaneSpec struct {
	// License is the Bindplane license key
	License string `json:"license"`

	// Mode specifies the operational mode(s) for Bindplane
	// +optional
	Mode []string `json:"mode,omitempty"`

	// Auth configuration for Bindplane
	// +optional
	Auth *AuthConfig `json:"auth,omitempty"`

	// Network configuration for Bindplane
	// +optional
	Network *NetworkConfig `json:"network,omitempty"`

	// Store configuration for Bindplane
	Store StoreConfig `json:"store"`

	// EventBus configuration for Bindplane
	// +optional
	EventBus *EventBusConfig `json:"eventBus,omitempty"`

	// Note: TransformAgent and Prometheus are implementation details and are not exposed
	// in the CRD. The controller will compute these values internally when generating
	// the Bindplane configuration. The type definitions below are kept for internal use.
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	// Type specifies the authentication type (e.g., "system")
	// +optional
	Type string `json:"type,omitempty"`

	// Username for authentication
	// +optional
	Username string `json:"username,omitempty"`

	// Password for authentication
	// +optional
	Password string `json:"password,omitempty"`

	// Note: sessionSecret is not exposed - it will be dynamically generated and stored as a Kubernetes secret
}

// NetworkConfig defines network configuration
type NetworkConfig struct {
	// Host specifies the bind address
	// +optional
	Host string `json:"host,omitempty"`

	// Port specifies the port to listen on
	// +optional
	Port string `json:"port,omitempty"`

	// RemoteURL specifies the remote URL
	// +optional
	RemoteURL string `json:"remoteURL,omitempty"`
}

// StoreConfig defines store configuration
type StoreConfig struct {
	// Type specifies the store type. Currently only "postgres" is supported.
	// +kubebuilder:validation:Enum=postgres
	Type string `json:"type"`

	// Postgres configuration (only used when type is "postgres")
	Postgres *PostgresConfig `json:"postgres"`
}

// PostgresConfig defines PostgreSQL store configuration
type PostgresConfig struct {
	// Host specifies the PostgreSQL host
	Host string `json:"host"`

	// Port specifies the PostgreSQL port
	// +optional
	Port string `json:"port,omitempty"`

	// ConnectTimeout specifies the connection timeout
	// +optional
	ConnectTimeout string `json:"connectTimeout,omitempty"`

	// StatementTimeout specifies the statement timeout
	// +optional
	StatementTimeout string `json:"statementTimeout,omitempty"`

	// Database specifies the database name
	// +optional
	Database string `json:"database,omitempty"`

	// SSLMode specifies the SSL mode
	// +optional
	SSLMode string `json:"sslmode,omitempty"`

	// Username specifies the PostgreSQL username
	// +optional
	Username string `json:"username,omitempty"`

	// Password specifies the PostgreSQL password
	// +optional
	Password string `json:"password,omitempty"`

	// MaxConnections specifies the maximum number of connections
	// +optional
	MaxConnections int `json:"maxConnections,omitempty"`

	// MaxLifetime specifies the maximum connection lifetime
	// +optional
	MaxLifetime string `json:"maxLifetime,omitempty"`

	// Schema specifies the database schema
	// +optional
	Schema string `json:"schema,omitempty"`
}

// EventBusConfig defines event bus configuration
type EventBusConfig struct {
	// Type specifies the event bus type. Supported values: "local", "nats", "googlePubSub", "azure"
	// +kubebuilder:validation:Enum=local;nats;googlePubSub;azure
	Type string `json:"type"`
}

// TransformAgentConfig defines transform agent configuration for internal use.
// This type is not exposed in the CRD spec - it's an implementation detail that the
// controller uses when generating the Bindplane configuration file.
// Note: enableRemote will always be set to true by the controller.
type TransformAgentConfig struct {
	// EnableRemote will always be true (set by controller)
	EnableRemote bool
	// RemoteAgents specifies the list of remote agent endpoints
	RemoteAgents []string
}

// PrometheusConfig defines Prometheus configuration for internal use.
// This type is not exposed in the CRD spec - it's an implementation detail that the
// controller uses when generating the Bindplane configuration file.
// Note: enableRemote will always be set to true by the controller.
type PrometheusConfig struct {
	// EnableRemote will always be true (set by controller)
	EnableRemote bool
	// Host specifies the Prometheus host
	Host string

	// Port specifies the Prometheus port
	Port string

	// RemoteWrite configuration
	RemoteWrite *PrometheusRemoteWriteConfig
}

// PrometheusRemoteWriteConfig defines remote write configuration for Prometheus
type PrometheusRemoteWriteConfig struct {
	// Endpoint specifies the remote write endpoint
	Endpoint string
}

// BindplaneStatus defines the observed state of Bindplane.
type BindplaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Bindplane is the Schema for the bindplanes API.
type Bindplane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BindplaneSpec   `json:"spec,omitempty"`
	Status BindplaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BindplaneList contains a list of Bindplane.
type BindplaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bindplane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bindplane{}, &BindplaneList{})
}
