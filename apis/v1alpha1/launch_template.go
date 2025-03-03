// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package v1alpha1

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LaunchTemplateSpec defines the desired state of LaunchTemplate.
//
// Describes a launch template.
type LaunchTemplateSpec struct {
	DefaultVersionNumber *int64 `json:"defaultVersionNumber,omitempty"`
	// The information for the launch template.
	// +kubebuilder:validation:Required
	LaunchTemplateData *RequestLaunchTemplateData `json:"launchTemplateData"`
	// A name for the launch template.
	// +kubebuilder:validation:Required
	Name *string `json:"name"`
	// The tags. The value parameter is required, but if you don't want the tag
	// to have a value, specify the parameter with no value, and we set the value
	// to an empty string.
	Tags []*Tag `json:"tags,omitempty"`
	// A description for the first version of the launch template.
	VersionDescription *string `json:"versionDescription,omitempty"`
}

// LaunchTemplateStatus defines the observed state of LaunchTemplate
type LaunchTemplateStatus struct {
	// All CRs managed by ACK have a common `Status.ACKResourceMetadata` member
	// that is used to contain resource sync state, account ownership,
	// constructed ARN for the resource
	// +kubebuilder:validation:Optional
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	// All CRs managed by ACK have a common `Status.Conditions` member that
	// contains a collection of `ackv1alpha1.Condition` objects that describe
	// the various terminal states of the CR and its backend AWS service API
	// resource
	// +kubebuilder:validation:Optional
	Conditions []*ackv1alpha1.Condition `json:"conditions"`
	// The time launch template was created.
	// +kubebuilder:validation:Optional
	CreateTime *metav1.Time `json:"createTime,omitempty"`
	// The principal that created the launch template.
	// +kubebuilder:validation:Optional
	CreatedBy *string `json:"createdBy,omitempty"`
	// The version number of the latest version of the launch template.
	// +kubebuilder:validation:Optional
	LatestVersionNumber *int64 `json:"latestVersionNumber,omitempty"`
	// The ID of the launch template.
	// +kubebuilder:validation:Optional
	LaunchTemplateID *string `json:"launchTemplateID,omitempty"`
	// The entity that manages the launch template.
	// +kubebuilder:validation:Optional
	Operator *OperatorResponse `json:"operator,omitempty"`
}

// LaunchTemplate is the Schema for the LaunchTemplates API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DefaultVersion",type=integer,priority=0,JSONPath=`.spec.defaultVersionNumber`
// +kubebuilder:printcolumn:name="LatestVersion",type=integer,priority=0,JSONPath=`.status.latestVersionNumber`
// +kubebuilder:printcolumn:name="LaunchTemplateID",type=string,priority=0,JSONPath=`.status.launchTemplateID`
type LaunchTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              LaunchTemplateSpec   `json:"spec,omitempty"`
	Status            LaunchTemplateStatus `json:"status,omitempty"`
}

// LaunchTemplateList contains a list of LaunchTemplate
// +kubebuilder:object:root=true
type LaunchTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LaunchTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LaunchTemplate{}, &LaunchTemplateList{})
}
