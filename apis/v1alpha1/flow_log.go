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

// FlowLogSpec defines the desired state of FlowLog.
//
// Describes a flow log.
type FlowLogSpec struct {

	// The ARN of the IAM role that allows Amazon EC2 to publish flow logs to the
	// log destination.
	//
	// This parameter is required if the destination type is cloud-watch-logs, or
	// if the destination type is kinesis-data-firehose and the delivery stream
	// and the resources to monitor are in different accounts.
	DeliverLogsPermissionARN *string `json:"deliverLogsPermissionARN,omitempty"`
	// The destination options.
	DestinationOptions *DestinationOptionsRequest `json:"destinationOptions,omitempty"`
	// The destination for the flow log data. The meaning of this parameter depends
	// on the destination type.
	//
	//   - If the destination type is cloud-watch-logs, specify the ARN of a CloudWatch
	//     Logs log group. For example: arn:aws:logs:region:account_id:log-group:my_group
	//     Alternatively, use the LogGroupName parameter.
	//
	//   - If the destination type is s3, specify the ARN of an S3 bucket. For
	//     example: arn:aws:s3:::my_bucket/my_subfolder/ The subfolder is optional.
	//     Note that you can't use AWSLogs as a subfolder name.
	//
	//   - If the destination type is kinesis-data-firehose, specify the ARN of
	//     a Kinesis Data Firehose delivery stream. For example: arn:aws:firehose:region:account_id:deliverystream:my_stream
	LogDestination *string `json:"logDestination,omitempty"`
	// The type of destination for the flow log data.
	//
	// Default: cloud-watch-logs
	LogDestinationType *string `json:"logDestinationType,omitempty"`
	// The fields to include in the flow log record. List the fields in the order
	// in which they should appear. If you omit this parameter, the flow log is
	// created using the default format. If you specify this parameter, you must
	// include at least one field. For more information about the available fields,
	// see Flow log records (https://docs.aws.amazon.com/vpc/latest/userguide/flow-log-records.html)
	// in the Amazon VPC User Guide or Transit Gateway Flow Log records (https://docs.aws.amazon.com/vpc/latest/tgw/tgw-flow-logs.html#flow-log-records)
	// in the Amazon Web Services Transit Gateway Guide.
	//
	// Specify the fields using the ${field-id} format, separated by spaces.
	LogFormat *string `json:"logFormat,omitempty"`
	// The name of a new or existing CloudWatch Logs log group where Amazon EC2
	// publishes your flow logs.
	//
	// This parameter is valid only if the destination type is cloud-watch-logs.
	LogGroupName *string `json:"logGroupName,omitempty"`
	// The maximum interval of time during which a flow of packets is captured and
	// aggregated into a flow log record. The possible values are 60 seconds (1
	// minute) or 600 seconds (10 minutes). This parameter must be 60 seconds for
	// transit gateway resource types.
	//
	// When a network interface is attached to a Nitro-based instance (https://docs.aws.amazon.com/ec2/latest/instancetypes/ec2-nitro-instances.html),
	// the aggregation interval is always 60 seconds or less, regardless of the
	// value that you specify.
	//
	// Default: 600
	MaxAggregationInterval *int64 `json:"maxAggregationInterval,omitempty"`
	// +kubebuilder:validation:Required
	ResourceID *string `json:"resourceID"`
	// The type of resource to monitor.
	// +kubebuilder:validation:Required
	ResourceType *string `json:"resourceType"`
	// The tags. The value parameter is required, but if you don't want the tag
	// to have a value, specify the parameter with no value, and we set the value
	// to an empty string.
	Tags []*Tag `json:"tags,omitempty"`
	// The type of traffic to monitor (accepted traffic, rejected traffic, or all
	// traffic). This parameter is not supported for transit gateway resource types.
	// It is required for the other resource types.
	TrafficType *string `json:"trafficType,omitempty"`
}

// FlowLogStatus defines the observed state of FlowLog
type FlowLogStatus struct {
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
	// Unique, case-sensitive identifier that you provide to ensure the idempotency
	// of the request.
	// +kubebuilder:validation:Optional
	ClientToken *string `json:"clientToken,omitempty"`
	// +kubebuilder:validation:Optional
	FlowLogID *string `json:"flowLogID,omitempty"`
	// Information about the flow logs that could not be created successfully.
	// +kubebuilder:validation:Optional
	Unsuccessful []*UnsuccessfulItem `json:"unsuccessful,omitempty"`
}

// FlowLog is the Schema for the FlowLogs API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type FlowLog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FlowLogSpec   `json:"spec,omitempty"`
	Status            FlowLogStatus `json:"status,omitempty"`
}

// FlowLogList contains a list of FlowLog
// +kubebuilder:object:root=true
type FlowLogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlowLog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FlowLog{}, &FlowLogList{})
}
