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

// DhcpOptionsSpec defines the desired state of DhcpOptions.
//
// The set of DHCP options.
type DHCPOptionsSpec struct {

	// A DHCP configuration option.

	// +kubebuilder:validation:Required

	DHCPConfigurations []*NewDHCPConfiguration `json:"dhcpConfigurations"`
	// The tags. The value parameter is required, but if you don't want the tag
	// to have a value, specify the parameter with no value, and we set the value
	// to an empty string.

	Tags []*Tag `json:"tags,omitempty"`

	VPC []*string `json:"vpc,omitempty"`

	VPCRefs []*ackv1alpha1.AWSResourceReferenceWrapper `json:"vpcRefs,omitempty"`
}

// DHCPOptionsStatus defines the observed state of DHCPOptions
type DHCPOptionsStatus struct {
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
	// The ID of the set of DHCP options.
	// +kubebuilder:validation:Optional
	DHCPOptionsID *string `json:"dhcpOptionsID,omitempty"`
	// The ID of the Amazon Web Services account that owns the DHCP options set.
	// +kubebuilder:validation:Optional
	OwnerID *string `json:"ownerID,omitempty"`
}

// DHCPOptions is the Schema for the DHCPOptions API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type=string,priority=0,JSONPath=`.status.dhcpOptionsID`
type DHCPOptions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DHCPOptionsSpec   `json:"spec,omitempty"`
	Status            DHCPOptionsStatus `json:"status,omitempty"`
}

// DHCPOptionsList contains a list of DHCPOptions
// +kubebuilder:object:root=true
type DHCPOptionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DHCPOptions `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DHCPOptions{}, &DHCPOptionsList{})
}
