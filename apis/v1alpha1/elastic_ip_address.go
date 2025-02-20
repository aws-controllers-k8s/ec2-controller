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

// ElasticIPAddressSpec defines the desired state of ElasticIPAddress.
type ElasticIPAddressSpec struct {

	// The Elastic IP address to recover or an IPv4 address from an address pool.
	Address *string `json:"address,omitempty"`
	// The ID of a customer-owned address pool. Use this parameter to let Amazon
	// EC2 select an address from the address pool. Alternatively, specify a specific
	// address from the address pool.
	CustomerOwnedIPv4Pool *string `json:"customerOwnedIPv4Pool,omitempty"`
	// A unique set of Availability Zones, Local Zones, or Wavelength Zones from
	// which Amazon Web Services advertises IP addresses. Use this parameter to
	// limit the IP address to this location. IP addresses cannot move between network
	// border groups.
	NetworkBorderGroup *string `json:"networkBorderGroup,omitempty"`
	// The ID of an address pool that you own. Use this parameter to let Amazon
	// EC2 select an address from the address pool. To specify a specific address
	// from the address pool, use the Address parameter instead.
	PublicIPv4Pool *string `json:"publicIPv4Pool,omitempty"`
	// The tags. The value parameter is required, but if you don't want the tag
	// to have a value, specify the parameter with no value, and we set the value
	// to an empty string.
	Tags []*Tag `json:"tags,omitempty"`
}

// ElasticIPAddressStatus defines the observed state of ElasticIPAddress
type ElasticIPAddressStatus struct {
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
	// The ID that represents the allocation of the Elastic IP address.
	// +kubebuilder:validation:Optional
	AllocationID *string `json:"allocationID,omitempty"`
	// The carrier IP address. This option is only available for network interfaces
	// that reside in a subnet in a Wavelength Zone.
	// +kubebuilder:validation:Optional
	CarrierIP *string `json:"carrierIP,omitempty"`
	// The customer-owned IP address.
	// +kubebuilder:validation:Optional
	CustomerOwnedIP *string `json:"customerOwnedIP,omitempty"`
	// The Elastic IP address.
	// +kubebuilder:validation:Optional
	PublicIP *string `json:"publicIP,omitempty"`
}

// ElasticIPAddress is the Schema for the ElasticIPAddresses API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ALLOCATION-ID",type=string,priority=0,JSONPath=`.status.allocationID`
// +kubebuilder:printcolumn:name="PUBLIC-IP",type=string,priority=0,JSONPath=`.status.publicIP`
type ElasticIPAddress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ElasticIPAddressSpec   `json:"spec,omitempty"`
	Status            ElasticIPAddressStatus `json:"status,omitempty"`
}

// ElasticIPAddressList contains a list of ElasticIPAddress
// +kubebuilder:object:root=true
type ElasticIPAddressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticIPAddress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticIPAddress{}, &ElasticIPAddressList{})
}
