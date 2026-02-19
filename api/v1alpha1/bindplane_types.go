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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BindplaneSpec defines the desired state of Bindplane.
type BindplaneSpec struct {
	// Config contains Bindplane's configuration (license, auth, network, store, eventBus)
	// This config is shared by Node, Jobs, and Jobs Migrate
	Config BindplaneConfigSpec `json:"config"`

	// Bindplane configuration and pod specification
	Bindplane BindplaneComponentSpec `json:"bindplane"`

	// Bindplane Jobs pod specification
	// +optional
	BindplaneJobs *BindplaneJobsComponentSpec `json:"bindplaneJobs,omitempty"`

	// Bindplane Jobs Migrate pod specification
	// +optional
	BindplaneJobsMigrate *BindplaneJobsMigrateComponentSpec `json:"bindplaneJobsMigrate,omitempty"`

	// Transform Agent pod specification
	// +optional
	TransformAgent *TransformAgentComponentSpec `json:"transformAgent,omitempty"`

	// Prometheus pod specification
	// +optional
	Prometheus *PrometheusComponentSpec `json:"prometheus,omitempty"`

	// NATS pod specification
	// +optional
	Nats *NatsComponentSpec `json:"nats,omitempty"`
}

// BindplaneComponentSpec defines the Bindplane component pod specification
type BindplaneComponentSpec struct {
	// Replicas specifies the number of replicas for Bindplane Node deployment
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// PodTemplate defines pod template specification for Bindplane Node
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// BindplaneJobsComponentSpec defines the Bindplane Jobs component pod specification
type BindplaneJobsComponentSpec struct {
	// PodTemplate defines pod template specification for Bindplane Jobs
	// Note: Jobs are restricted to 1 replica and cannot be scaled
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// BindplaneJobsMigrateComponentSpec defines the Bindplane Jobs Migrate component pod specification
type BindplaneJobsMigrateComponentSpec struct {
	// PodTemplate defines pod template specification for Bindplane Jobs Migrate
	// Note: Jobs Migrate are restricted to 1 replica and cannot be scaled
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// BindplaneConfigSpec defines Bindplane's configuration
type BindplaneConfigSpec struct {
	// License is the Bindplane license key
	License string `json:"license"`

	// Auth configuration for Bindplane
	// +optional
	Auth *AuthConfig `json:"auth,omitempty"`

	// Network configuration for Bindplane
	// +optional
	Network *NetworkConfig `json:"network,omitempty"`

	// Store configuration for Bindplane
	Store StoreConfig `json:"store"`
}

// TransformAgentComponentSpec defines the Transform Agent component pod specification
type TransformAgentComponentSpec struct {
	// Replicas specifies the number of replicas for Transform Agent deployment
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// PodTemplate defines pod template specification for Transform Agent
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// PrometheusComponentSpec defines the Prometheus component pod specification
type PrometheusComponentSpec struct {
	// PodTemplate defines pod template specification for Prometheus
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// Storage defines the persistent storage configuration for Prometheus
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`
}

// NatsComponentSpec defines the NATS component pod specification
type NatsComponentSpec struct {
	// Replicas specifies the number of replicas for NATS StatefulSet
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// PodTemplate defines pod template specification for NATS
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// StorageSpec defines persistent storage configuration
type StorageSpec struct {
	// VolumeClaimTemplate defines the template for creating PersistentVolumeClaims
	// This follows the same structure as StatefulSet volumeClaimTemplates
	VolumeClaimTemplate *VolumeClaimTemplate `json:"volumeClaimTemplate,omitempty"`
}

// VolumeClaimTemplate defines a template for creating PersistentVolumeClaims
type VolumeClaimTemplate struct {
	// Metadata for the PersistentVolumeClaim
	// +optional
	Metadata *metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the PersistentVolumeClaim specification
	Spec corev1.PersistentVolumeClaimSpec `json:"spec"`
}

// PodTemplateSpec defines pod template specification.
// This embeds corev1.PodTemplateSpec to allow arbitrary pod spec fields.
// Note: The operator will merge this with operator-managed fields, ensuring
// operator-managed fields (like ServiceAccountName, containers, etc.) take precedence.
// +kubebuilder:pruning:PreserveUnknownFields
type PodTemplateSpec struct {
	// Embedded PodTemplateSpec allows users to specify arbitrary pod spec fields
	// such as affinity, tolerations, nodeSelector, securityContext, etc.
	// Operator-managed fields (ServiceAccountName, containers, etc.) will be preserved.
	corev1.PodTemplateSpec `json:",inline"`
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

// BindplaneStatus defines the observed state of Bindplane.
type BindplaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=bindplanes,singular=bindplane,scope=Namespaced

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
