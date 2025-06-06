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

// CapacityReservationSpec defines the desired state of CapacityReservation.
//
// Describes a Capacity Reservation.
type CapacityReservationSpec struct {

	// Reserved for future use.
	AdditionalInfo *string `json:"additionalInfo,omitempty"`
	// The Availability Zone in which to create the Capacity Reservation.
	AvailabilityZone *string `json:"availabilityZone,omitempty"`
	// The ID of the Availability Zone in which to create the Capacity Reservation.
	AvailabilityZoneID *string `json:"availabilityZoneID,omitempty"`
	// Required for future-dated Capacity Reservations only. To create a Capacity
	// Reservation for immediate use, omit this parameter.
	//
	// Specify a commitment duration, in seconds, for the future-dated Capacity
	// Reservation.
	//
	// The commitment duration is a minimum duration for which you commit to having
	// the future-dated Capacity Reservation in the active state in your account
	// after it has been delivered.
	//
	// For more information, see Commitment duration (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/cr-concepts.html#cr-commitment-duration).
	CommitmentDuration *int64 `json:"commitmentDuration,omitempty"`
	// Required for future-dated Capacity Reservations only. To create a Capacity
	// Reservation for immediate use, omit this parameter.
	//
	// Indicates that the requested capacity will be delivered in addition to any
	// running instances or reserved capacity that you have in your account at the
	// requested date and time.
	//
	// The only supported value is incremental.
	DeliveryPreference *string `json:"deliveryPreference,omitempty"`
	// Indicates whether the Capacity Reservation supports EBS-optimized instances.
	// This optimization provides dedicated throughput to Amazon EBS and an optimized
	// configuration stack to provide optimal I/O performance. This optimization
	// isn't available with all instance types. Additional usage charges apply when
	// using an EBS- optimized instance.
	EBSOptimized *bool `json:"ebsOptimized,omitempty"`
	// The date and time at which the Capacity Reservation expires. When a Capacity
	// Reservation expires, the reserved capacity is released and you can no longer
	// launch instances into it. The Capacity Reservation's state changes to expired
	// when it reaches its end date and time.
	//
	// You must provide an EndDate value if EndDateType is limited. Omit EndDate
	// if EndDateType is unlimited.
	//
	// If the EndDateType is limited, the Capacity Reservation is cancelled within
	// an hour from the specified time. For example, if you specify 5/31/2019, 13:30:55,
	// the Capacity Reservation is guaranteed to end between 13:30:55 and 14:30:55
	// on 5/31/2019.
	//
	// If you are requesting a future-dated Capacity Reservation, you can't specify
	// an end date and time that is within the commitment duration.
	EndDate *metav1.Time `json:"endDate,omitempty"`
	// Indicates the way in which the Capacity Reservation ends. A Capacity Reservation
	// can have one of the following end types:
	//
	//   - unlimited - The Capacity Reservation remains active until you explicitly
	//     cancel it. Do not provide an EndDate if the EndDateType is unlimited.
	//
	//   - limited - The Capacity Reservation expires automatically at a specified
	//     date and time. You must provide an EndDate value if the EndDateType value
	//     is limited.
	EndDateType *string `json:"endDateType,omitempty"`
	// Deprecated.
	EphemeralStorage *bool `json:"ephemeralStorage,omitempty"`
	// The number of instances for which to reserve capacity.
	//
	// You can request future-dated Capacity Reservations for an instance count
	// with a minimum of 100 VPUs. For example, if you request a future-dated Capacity
	// Reservation for m5.xlarge instances, you must request at least 25 instances
	// (25 * m5.xlarge = 100 vCPUs).
	//
	// Valid range: 1 - 1000
	// +kubebuilder:validation:Required
	InstanceCount *int64 `json:"instanceCount"`
	// Indicates the type of instance launches that the Capacity Reservation accepts.
	// The options include:
	//
	//   - open - The Capacity Reservation automatically matches all instances
	//     that have matching attributes (instance type, platform, and Availability
	//     Zone). Instances that have matching attributes run in the Capacity Reservation
	//     automatically without specifying any additional parameters.
	//
	//   - targeted - The Capacity Reservation only accepts instances that have
	//     matching attributes (instance type, platform, and Availability Zone),
	//     and explicitly target the Capacity Reservation. This ensures that only
	//     permitted instances can use the reserved capacity.
	//
	// If you are requesting a future-dated Capacity Reservation, you must specify
	// targeted.
	//
	// Default: open
	InstanceMatchCriteria *string `json:"instanceMatchCriteria,omitempty"`
	// The type of operating system for which to reserve capacity.
	// +kubebuilder:validation:Required
	InstancePlatform *string `json:"instancePlatform"`
	// The instance type for which to reserve capacity.
	//
	// You can request future-dated Capacity Reservations for instance types in
	// the C, M, R, I, and T instance families only.
	//
	// For more information, see Instance types (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html)
	// in the Amazon EC2 User Guide.
	// +kubebuilder:validation:Required
	InstanceType *string `json:"instanceType"`
	// Not supported for future-dated Capacity Reservations.
	//
	// The Amazon Resource Name (ARN) of the Outpost on which to create the Capacity
	// Reservation.
	//
	// Regex Pattern: `^arn:aws([a-z-]+)?:outposts:[a-z\d-]+:\d{12}:outpost/op-[a-f0-9]{17}$`
	OutpostARN *string `json:"outpostARN,omitempty"`
	// Not supported for future-dated Capacity Reservations.
	//
	// The Amazon Resource Name (ARN) of the cluster placement group in which to
	// create the Capacity Reservation. For more information, see Capacity Reservations
	// for cluster placement groups (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/cr-cpg.html)
	// in the Amazon EC2 User Guide.
	//
	// Regex Pattern: `^arn:aws([a-z-]+)?:ec2:[a-z\d-]+:\d{12}:placement-group/^.{1,255}$`
	PlacementGroupARN *string `json:"placementGroupARN,omitempty"`
	// Required for future-dated Capacity Reservations only. To create a Capacity
	// Reservation for immediate use, omit this parameter.
	//
	// The date and time at which the future-dated Capacity Reservation should become
	// available for use, in the ISO8601 format in the UTC time zone (YYYY-MM-DDThh:mm:ss.sssZ).
	//
	// You can request a future-dated Capacity Reservation between 5 and 120 days
	// in advance.
	StartDate *metav1.Time `json:"startDate,omitempty"`
	// The tags. The value parameter is required, but if you don't want the tag
	// to have a value, specify the parameter with no value, and we set the value
	// to an empty string.
	Tags []*Tag `json:"tags,omitempty"`
	// Indicates the tenancy of the Capacity Reservation. A Capacity Reservation
	// can have one of the following tenancy settings:
	//
	//   - default - The Capacity Reservation is created on hardware that is shared
	//     with other Amazon Web Services accounts.
	//
	//   - dedicated - The Capacity Reservation is created on single-tenant hardware
	//     that is dedicated to a single Amazon Web Services account.
	Tenancy *string `json:"tenancy,omitempty"`
}

// CapacityReservationStatus defines the observed state of CapacityReservation
type CapacityReservationStatus struct {
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
	// The remaining capacity. Indicates the number of instances that can be launched
	// in the Capacity Reservation.
	// +kubebuilder:validation:Optional
	AvailableInstanceCount *int64 `json:"availableInstanceCount,omitempty"`
	// Information about instance capacity usage.
	// +kubebuilder:validation:Optional
	CapacityAllocations []*CapacityAllocation `json:"capacityAllocations,omitempty"`
	// The ID of the Capacity Reservation Fleet to which the Capacity Reservation
	// belongs. Only valid for Capacity Reservations that were created by a Capacity
	// Reservation Fleet.
	// +kubebuilder:validation:Optional
	CapacityReservationFleetID *string `json:"capacityReservationFleetID,omitempty"`
	// The ID of the Capacity Reservation.
	// +kubebuilder:validation:Optional
	CapacityReservationID *string `json:"capacityReservationID,omitempty"`
	// Information about your commitment for a future-dated Capacity Reservation.
	// +kubebuilder:validation:Optional
	CommitmentInfo *CapacityReservationCommitmentInfo `json:"commitmentInfo,omitempty"`
	// The date and time at which the Capacity Reservation was created.
	// +kubebuilder:validation:Optional
	CreateDate *metav1.Time `json:"createDate,omitempty"`
	// The ID of the Amazon Web Services account that owns the Capacity Reservation.
	// +kubebuilder:validation:Optional
	OwnerID *string `json:"ownerID,omitempty"`
	// The type of Capacity Reservation.
	// +kubebuilder:validation:Optional
	ReservationType *string `json:"reservationType,omitempty"`
	// The current state of the Capacity Reservation. A Capacity Reservation can
	// be in one of the following states:
	//
	//    * active - The capacity is available for use.
	//
	//    * expired - The Capacity Reservation expired automatically at the date
	//    and time specified in your reservation request. The reserved capacity
	//    is no longer available for your use.
	//
	//    * cancelled - The Capacity Reservation was canceled. The reserved capacity
	//    is no longer available for your use.
	//
	//    * pending - The Capacity Reservation request was successful but the capacity
	//    provisioning is still pending.
	//
	//    * failed - The Capacity Reservation request has failed. A request can
	//    fail due to request parameters that are not valid, capacity constraints,
	//    or instance limit constraints. You can view a failed request for 60 minutes.
	//
	//    * scheduled - (Future-dated Capacity Reservations only) The future-dated
	//    Capacity Reservation request was approved and the Capacity Reservation
	//    is scheduled for delivery on the requested start date.
	//
	//    * assessing - (Future-dated Capacity Reservations only) Amazon EC2 is
	//    assessing your request for a future-dated Capacity Reservation.
	//
	//    * delayed - (Future-dated Capacity Reservations only) Amazon EC2 encountered
	//    a delay in provisioning the requested future-dated Capacity Reservation.
	//    Amazon EC2 is unable to deliver the requested capacity by the requested
	//    start date and time.
	//
	//    * unsupported - (Future-dated Capacity Reservations only) Amazon EC2 can't
	//    support the future-dated Capacity Reservation request due to capacity
	//    constraints. You can view unsupported requests for 30 days. The Capacity
	//    Reservation will not be delivered.
	// +kubebuilder:validation:Optional
	State *string `json:"state,omitempty"`
	// The total number of instances for which the Capacity Reservation reserves
	// capacity.
	// +kubebuilder:validation:Optional
	TotalInstanceCount *int64 `json:"totalInstanceCount,omitempty"`
	// The ID of the Amazon Web Services account to which billing of the unused
	// capacity of the Capacity Reservation is assigned.
	//
	// Regex Pattern: `^[0-9]{12}$`
	// +kubebuilder:validation:Optional
	UnusedReservationBillingOwnerID *string `json:"unusedReservationBillingOwnerID,omitempty"`
}

// CapacityReservation is the Schema for the CapacityReservations API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type=string,priority=0,JSONPath=`.status.capacityReservationID`
// +kubebuilder:printcolumn:name="START_DATE",type=date,priority=0,JSONPath=`.spec.startDate`
// +kubebuilder:printcolumn:name="STATE",type=string,priority=0,JSONPath=`.status.state`
type CapacityReservation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CapacityReservationSpec   `json:"spec,omitempty"`
	Status            CapacityReservationStatus `json:"status,omitempty"`
}

// CapacityReservationList contains a list of CapacityReservation
// +kubebuilder:object:root=true
type CapacityReservationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CapacityReservation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CapacityReservation{}, &CapacityReservationList{})
}
