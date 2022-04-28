/*
Copyright AppsCode Inc. and Contributors

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

const (
	ResourceKindMetricsConfiguration = "MetricsConfiguration"
	ResourceMetricsConfiguration     = "metricsconfiguration"
	ResourceMetricsConfigurations    = "metricsconfigurations"
)

// MetricsConfiguration defines a generic metrics configuration
// in prometheus style for a specific resource object.

// +genclient
// +genclient:nonNamespaced
// +genclient:skipVerbs=updateStatus
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=metricsconfigurations,singular=metricsconfiguration,scope=Cluster,categories={metrics,appscode,all}
// +kubebuilder:printcolumn:name="APIVersion",type="string",JSONPath=".spec.targetRef.apiVersion"
// +kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.targetRef.kind"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type MetricsConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              MetricsConfigurationSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// MetricsConfigurationSpec is the spec of MetricsConfiguration object.
type MetricsConfigurationSpec struct {
	// TargetRef defines the object for which metrics will be collected
	TargetRef TargetRef `json:"targetRef" protobuf:"bytes,1,opt,name=targetRef"`

	// CommonLabels defines the common labels added to all the exported metrics
	// +optional
	CommonLabels []Label `json:"commonLabels,omitempty" protobuf:"bytes,2,rep,name=commonLabels"`

	// List of Metrics configuration for the resource object defined in TargetRef
	Metrics []Metrics `json:"metrics" protobuf:"bytes,3,rep,name=metrics"`
}

// TargetRef contains the Object's apiVersion & kind to specify the target resource
type TargetRef struct {
	// APIVersion defines the versioned schema of this representation of an object.
	APIVersion string `json:"apiVersion" protobuf:"bytes,1,opt,name=apiVersion"`

	// Kind is a string value representing the REST resource this object represents.
	// In CamelCase.
	Kind string `json:"kind" protobuf:"bytes,2,opt,name=kind"`
}

// Metrics contains the configuration of a metric in prometheus style.
type Metrics struct {
	// Name defines the metrics name. Name should be in snake case.
	// Example: kube_deployment_spec_replicas
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Help is used to describe the metrics.
	// Example: For kube_deployment_spec_replicas, help string can be "Number of desired pods for a deployment."
	Help string `json:"help" protobuf:"bytes,2,opt,name=help"`

	// Type defines the metrics type.
	// For kubernetes based object, types can only be "gauge"
	// +kubebuilder:validation:Enum=gauge;
	Type string `json:"type" protobuf:"bytes,3,opt,name=type"`

	// Field defines the metric value path of the manifest file and the type of that value
	// +optional
	Field Field `json:"field,omitempty" protobuf:"bytes,4,opt,name=field"`

	// Labels defines the metric labels as a key-value pair
	// +optional
	Labels []Label `json:"labels,omitempty" protobuf:"bytes,5,rep,name=labels"`

	// Params is list of parameters configuration used in expression evaluation
	// +optional
	Params []Parameter `json:"params,omitempty" protobuf:"bytes,6,rep,name=params"`

	// States handle metrics with label cardinality.
	// States specify the possible states for a label
	// and their corresponding MetricValue configuration.
	//
	// Metrics must contain either States or MetricValue.
	// If both are specified, MetricValue will be ignored.
	// +optional
	States *State `json:"states,omitempty" protobuf:"bytes,7,opt,name=states"`

	// MetricValue defines the configuration to obtain metric value.
	//
	// Metrics must contain either States or MetricValue.
	// If both are specified, MetricValue will be ignored.
	// +optional
	MetricValue MetricValue `json:"metricValue,omitempty" protobuf:"bytes,8,opt,name=metricValue"`
}

type FieldType string

const (
	FieldTypeInteger  FieldType = "Integer"
	FieldTypeDateTime FieldType = "DateTime"
	FieldTypeArray    FieldType = "Array"
	FieldTypeString   FieldType = "String"
)

// Field contains the information of the field for which metric is collected.
// Example: To collect available replica count for a Deployment, Field's Path
// will be .statue.availableReplicas and the Type will be Integer.
//
// When some labels are collected with metric value 1 and
// the values are not from an array then Field can be skipped
// or will be ignored if specified. Otherwise Field must be specified.
type Field struct {
	// Path defines the json path of the object.
	// Example: For deployment spec replica count, the path will be .spec.replicas
	Path string `json:"path" protobuf:"bytes,1,opt,name=path"`

	// Type defines the type of the value in the given Path
	// Type can be "Integer" for integer value like .spec.replicas,
	// "DateTime" for time stamp value like .metadata.creationTimestamp
	// "Array" for array field like .spec.containers
	// "String" for string field like .statue.phase (for pod status)
	// +kubebuilder:validation:Enum=Integer;DateTime;Array;String
	Type FieldType `json:"type" protobuf:"bytes,2,opt,name=type,casttype=FieldType"`
}

// Label contains the information of a metric label.
// Given labels are always added in the metrics along with resource name and namespace.
// Resource's name and namespace are always added in the labels by default.
// No configuration is needed for name and namespace labels.
//
// Example: kube_pod_info{pod="<pod_name>", namespace="<pod_namespace>", host_ip="172.18.0.2", pod_ip="10.244.0.14", node="kind-control-plane"}  1
// In the example pod, namespace, host_ip, pod_ip, node are labels.
// pod(resource name) and namespace are default labels. No configurations is needed for those.
//
// To generate others labels, config should be given in the following way
//
// labels:
//   - key: host_ip
//     valuePath: .status.hostIP
//   - key: pod_ip
//     valuePath: .status.podIP
//   - key: node
//     valuePath: .spec.nodeName
//
// Either Value or ValuePath must be specified for a Label.
// If both is specified, ValuePath is ignored.
// Note that if a valuePath doesn't exist for a label key, the label will be ignored.
type Label struct {
	// Key defines the label key
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`

	// Value defines the hard coded label value.
	// Example:
	// labels:
	//   - key: unit
	//     value: byte
	//   - key: environment
	//     value: production
	//
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`

	// ValuePath defines the label value path.
	// Example: To add deployment's resource version as labels,
	// labels:
	//   - key: version
	//     valuePath: .metadata.resourceVersion
	//
	// +optional
	ValuePath string `json:"valuePath" protobuf:"bytes,3,opt,name=valuePath"`
}

// Parameter contains the information of a parameter used in expression evaluation
// Parameter should contain an user defined key and corresponding Value or ValuePath.
// Either Value or ValuePath must be specified.
// If both are specified, ValuePath is ignored.
type Parameter struct {
	// Key defines the parameter's key
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`

	// Value defines user defined parameter's value.
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`

	// ValuePath defines the manifest field path for the parameter's value.
	// Example: To add deployment's spec replica count as parameter,
	// params:
	//   - key: replica
	//     valuePath: .spec.replicas
	// +optional
	ValuePath string `json:"valuePath,omitempty" protobuf:"bytes,3,opt,name=valuePath"`
}

// State contains the configuration for generating all the time series
// of a metric with label cardinality is greater than 1.
//
// Example: kube_pod_status_phase has a label called "phase" which value can be
// "Running", "Succeeded", "Failed", "Unknown", "Pending".
// So the cardinality of label phase is equal to 5. So kube_pod_status_phase will
// always generate five time series for a single pod.
//
// For a pod which .status.phase=Running, the time series are:
// kube_pod_status_phase{...,phase="Running",...} 1
// kube_pod_status_phase{...,phase="Succeeded",...} 0
// kube_pod_status_phase{...,phase="Failed",...} 0
// kube_pod_status_phase{...,phase="Unknown",...} 0
// kube_pod_status_phase{...,phase="Pending",...} 0
type State struct {
	// LabelKey defines an user defined label key of the label
	// which label cardinality is greater than one.
	// Example: For metric "kube_pod_status_phase", the LabelKey can be "phase"
	LabelKey string `json:"labelKey" protobuf:"bytes,1,opt,name=labelKey"`

	// Values contains the list of state values.
	// The size of the list is always equal to the cardinality of that label.
	// Example: "kube_pod_statue_phase" metric has a label "phase"
	// which cardinality is equal to 5. So Values should have StateValues config for all of them.
	Values []StateValues `json:"values" protobuf:"bytes,2,rep,name=values"`
}

// StateValues contains the information of a state value.
// StateValues is used to define state with all possible
// label values and corresponding MetricValue.
type StateValues struct {
	// LabelValue defines the value of the label.
	// Example: For labelKey "phase" (metric: kube_pod_status_phase path: .status.phase )
	// label value can be "Running", "Succeeded", "Failed", "Unknown" and "Pending"
	LabelValue string `json:"labelValue" protobuf:"bytes,1,opt,name=labelValue"`

	// MetricValue defines the configuration of the metric value for the corresponding LabelValue
	MetricValue MetricValue `json:"metricValue" protobuf:"bytes,2,opt,name=metricValue"`
}

// MetricValue contains the configuration to obtain the value for a metric.
// Note that MetricValue should contain only one field: Value or ValueFromPath or ValueFromExpression.
// If multiple fields are assigned then only one field is considered and other fields are ignored.
// The priority rule is Value > ValueFromPath > ValueFromExpression.
type MetricValue struct {
	// Value contains the metric value. It is always equal to 1.
	// It is defined when some information of the object is
	// collected as labels but there is no specific metric value.
	//
	// Example: For metrics "kube_pod_info", there are some information
	// like host_ip, pod_ip, node name is collected as labels.
	// As there must be a metric value, metric value is kept as 1.
	// The metric will look like `kube_pod_info{host_ip="172.18.0.2", pod_ip="10.244.0.14", node="kind-control-plane" .....}  1`
	// +optional
	Value *float64 `json:"value,omitempty" protobuf:"fixed64,1,opt,name=value"`

	// ValueFromPath contains the field path of the manifest file of a object.
	// ValueFromPath is used when the metric value is coming from
	// any specific json path of the object.
	//
	// Example: For metrics "kube_deployment_spec_replicas",
	// the metricValue is coming from a specific path .spec.replicas
	// In this case, valueFromPath: .spec.replicas
	// Some example of json path: .metadata.observedGeneration, .spec.restartPolicy, .status.startTime
	//
	// Some example of json path
	// which is coming from an element of an array: .spec.containers[*].image, .status.containerStatuses[*].restartCount
	// +optional
	ValueFromPath string `json:"valueFromPath,omitempty" protobuf:"bytes,2,opt,name=valueFromPath"`

	// ValueFromExpression contains an expression for the metric value
	// expression can be a function as well. Parameters is used in the expression string
	//
	// Available expression evaluation functions are:
	//
	// int() returns 1 if the expression is true otherwise 0,
	// example: int(phase == 'Running')
	//
	// percentage(percent, total, roundUp) returns the value of (percent * total%) when `percent` contains the percent(%) value.
	// If percent represents an Integer value, then it will simply return it.
	// roundUp is an optional field. By default, its value is false. If roundUp is set as `true`, the resultant value will be rounded up.
	// example: (i) percentage("25%", 4) will return 1.
	//         (ii) percentage("25%", 1 , true) will return 1 as roundUp is set as true.
	//        (iii) percentage(2, 4) will return 2 as percent is representing an Integer value.
	//
	// cpu_cores() returns the cpu in unit core
	// example: cpu_cores(cpu), for cpu value 150m, it will return 0.15
	//
	// bytes() returns the memory size in byte
	// example: bytes(memory), for memory value 1 ki, it will return 1024
	//
	// unix() returns the DateTime string into unix format.
	// example: unix(dateTime) will return the corresponding unix value for the given dateTime
	//
	// in above examples phase, replicas, maxUnavailable, cpu, memory, dateTime are Parameter's key
	// those values will come from corresponding Parameter's value
	//
	// Some expression evaluation functions are used for calculating resource requests and limits.
	// Those functions are stated here: https://github.com/kmodules/resource-metrics/blob/master/eval.go
	// +optional
	ValueFromExpression string `json:"valueFromExpression,omitempty" protobuf:"bytes,3,opt,name=valueFromExpression"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MetricsConfigurationList is a list of MetricsConfiguration
type MetricsConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []MetricsConfiguration `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
