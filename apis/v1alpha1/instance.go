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

// InstanceSpec defines the desired state of Instance.
//
// Describes an instance.
type InstanceSpec struct {
	// The block device mapping, which defines the EBS volumes and instance store
	// volumes to attach to the instance at launch. For more information, see Block
	// device mappings (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html)
	// in the Amazon EC2 User Guide.
	BlockDeviceMappings []*BlockDeviceMapping `json:"blockDeviceMappings,omitempty"`
	// Information about the Capacity Reservation targeting option. If you do not
	// specify this parameter, the instance's Capacity Reservation preference defaults
	// to open, which enables it to run in any open Capacity Reservation that has
	// matching attributes (instance type, platform, Availability Zone).
	CapacityReservationSpecification *CapacityReservationSpecification `json:"capacityReservationSpecification,omitempty"`
	// The CPU options for the instance. For more information, see Optimizing CPU
	// options (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-optimize-cpu.html)
	// in the Amazon EC2 User Guide.
	CPUOptions *CPUOptionsRequest `json:"cpuOptions,omitempty"`
	// The credit option for CPU usage of the burstable performance instance. Valid
	// values are standard and unlimited. To change this attribute after launch,
	// use ModifyInstanceCreditSpecification (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifyInstanceCreditSpecification.html).
	// For more information, see Burstable performance instances (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/burstable-performance-instances.html)
	// in the Amazon EC2 User Guide.
	//
	// Default: standard (T2 instances) or unlimited (T3/T3a instances)
	//
	// For T3 instances with host tenancy, only standard is supported.
	CreditSpecification *CreditSpecificationRequest `json:"creditSpecification,omitempty"`
	// If you set this parameter to true, you can't terminate the instance using
	// the Amazon EC2 console, CLI, or API; otherwise, you can. To change this attribute
	// after launch, use ModifyInstanceAttribute (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifyInstanceAttribute.html).
	// Alternatively, if you set InstanceInitiatedShutdownBehavior to terminate,
	// you can terminate the instance by running the shutdown command from the instance.
	//
	// Default: false
	DisableAPITermination *bool `json:"disableAPITermination,omitempty"`
	// Indicates whether the instance is optimized for Amazon EBS I/O. This optimization
	// provides dedicated throughput to Amazon EBS and an optimized configuration
	// stack to provide optimal Amazon EBS I/O performance. This optimization isn't
	// available with all instance types. Additional usage charges apply when using
	// an EBS-optimized instance.
	//
	// Default: false
	EBSOptimized *bool `json:"ebsOptimized,omitempty"`
	// An elastic GPU to associate with the instance. An Elastic GPU is a GPU resource
	// that you can attach to your Windows instance to accelerate the graphics performance
	// of your applications. For more information, see Amazon EC2 Elastic GPUs (https://docs.aws.amazon.com/AWSEC2/latest/WindowsGuide/elastic-graphics.html)
	// in the Amazon EC2 User Guide.
	ElasticGPUSpecification []*ElasticGPUSpecification `json:"elasticGPUSpecification,omitempty"`
	// An elastic inference accelerator to associate with the instance. Elastic
	// inference accelerators are a resource you can attach to your Amazon EC2 instances
	// to accelerate your Deep Learning (DL) inference workloads.
	//
	// You cannot specify accelerators from different generations in the same request.
	ElasticInferenceAccelerators []*ElasticInferenceAccelerator `json:"elasticInferenceAccelerators,omitempty"`
	// Indicates whether the instance is enabled for Amazon Web Services Nitro Enclaves.
	// For more information, see What is Amazon Web Services Nitro Enclaves? (https://docs.aws.amazon.com/enclaves/latest/user/nitro-enclave.html)
	// in the Amazon Web Services Nitro Enclaves User Guide.
	//
	// You can't enable Amazon Web Services Nitro Enclaves and hibernation on the
	// same instance.
	EnclaveOptions *EnclaveOptionsRequest `json:"enclaveOptions,omitempty"`
	// Indicates whether an instance is enabled for hibernation. For more information,
	// see Hibernate your instance (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Hibernate.html)
	// in the Amazon EC2 User Guide.
	//
	// You can't enable hibernation and Amazon Web Services Nitro Enclaves on the
	// same instance.
	HibernationOptions *HibernationOptionsRequest `json:"hibernationOptions,omitempty"`
	// The name or Amazon Resource Name (ARN) of an IAM instance profile.
	IAMInstanceProfile *IAMInstanceProfileSpecification `json:"iamInstanceProfile,omitempty"`
	// The ID of the AMI. An AMI ID is required to launch an instance and must be
	// specified here or in a launch template.
	ImageID *string `json:"imageID,omitempty"`
	// Indicates whether an instance stops or terminates when you initiate shutdown
	// from the instance (using the operating system command for system shutdown).
	//
	// Default: stop
	InstanceInitiatedShutdownBehavior *string `json:"instanceInitiatedShutdownBehavior,omitempty"`
	// The market (purchasing) option for the instances.
	//
	// For RunInstances, persistent Spot Instance requests are only supported when
	// InstanceInterruptionBehavior is set to either hibernate or stop.
	InstanceMarketOptions *InstanceMarketOptionsRequest `json:"instanceMarketOptions,omitempty"`
	// The instance type. For more information, see Instance types (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html)
	// in the Amazon EC2 User Guide.
	//
	// Default: m1.small
	InstanceType *string `json:"instanceType,omitempty"`
	// [EC2-VPC] The number of IPv6 addresses to associate with the primary network
	// interface. Amazon EC2 chooses the IPv6 addresses from the range of your subnet.
	// You cannot specify this option and the option to assign specific IPv6 addresses
	// in the same request. You can specify this option if you've specified a minimum
	// number of instances to launch.
	//
	// You cannot specify this option and the network interfaces option in the same
	// request.
	IPv6AddressCount *int64 `json:"ipv6AddressCount,omitempty"`
	// [EC2-VPC] The IPv6 addresses from the range of the subnet to associate with
	// the primary network interface. You cannot specify this option and the option
	// to assign a number of IPv6 addresses in the same request. You cannot specify
	// this option if you've specified a minimum number of instances to launch.
	//
	// You cannot specify this option and the network interfaces option in the same
	// request.
	IPv6Addresses []*InstanceIPv6Address `json:"ipv6Addresses,omitempty"`
	// The ID of the kernel.
	//
	// We recommend that you use PV-GRUB instead of kernels and RAM disks. For more
	// information, see PV-GRUB (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/UserProvidedkernels.html)
	// in the Amazon EC2 User Guide.
	KernelID *string `json:"kernelID,omitempty"`
	// The name of the key pair. You can create a key pair using CreateKeyPair (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateKeyPair.html)
	// or ImportKeyPair (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ImportKeyPair.html).
	//
	// If you do not specify a key pair, you can't connect to the instance unless
	// you choose an AMI that is configured to allow users another way to log in.
	KeyName *string `json:"keyName,omitempty"`
	// The launch template to use to launch the instances. Any parameters that you
	// specify in RunInstances override the same parameters in the launch template.
	// You can specify either the name or ID of a launch template, but not both.
	LaunchTemplate *LaunchTemplateSpecification `json:"launchTemplate,omitempty"`
	// The license configurations.
	LicenseSpecifications []*LicenseConfigurationRequest `json:"licenseSpecifications,omitempty"`
	// The maximum number of instances to launch. If you specify more instances
	// than Amazon EC2 can launch in the target Availability Zone, Amazon EC2 launches
	// the largest possible number of instances above MinCount.
	//
	// Constraints: Between 1 and the maximum number you're allowed for the specified
	// instance type. For more information about the default limits, and how to
	// request an increase, see How many instances can I run in Amazon EC2 (http://aws.amazon.com/ec2/faqs/#How_many_instances_can_I_run_in_Amazon_EC2)
	// in the Amazon EC2 FAQ.
	MaxCount *int64 `json:"maxCount,omitempty"`
	// The metadata options for the instance. For more information, see Instance
	// metadata and user data (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html).
	MetadataOptions *InstanceMetadataOptionsRequest `json:"metadataOptions,omitempty"`
	// The minimum number of instances to launch. If you specify a minimum that
	// is more instances than Amazon EC2 can launch in the target Availability Zone,
	// Amazon EC2 launches no instances.
	//
	// Constraints: Between 1 and the maximum number you're allowed for the specified
	// instance type. For more information about the default limits, and how to
	// request an increase, see How many instances can I run in Amazon EC2 (http://aws.amazon.com/ec2/faqs/#How_many_instances_can_I_run_in_Amazon_EC2)
	// in the Amazon EC2 General FAQ.
	MinCount *int64 `json:"minCount,omitempty"`
	// Specifies whether detailed monitoring is enabled for the instance.
	Monitoring *RunInstancesMonitoringEnabled `json:"monitoring,omitempty"`
	// The network interfaces to associate with the instance. If you specify a network
	// interface, you must specify any security groups and subnets as part of the
	// network interface.
	NetworkInterfaces []*InstanceNetworkInterfaceSpecification `json:"networkInterfaces,omitempty"`
	// The placement for the instance.
	Placement *Placement `json:"placement,omitempty"`
	// [EC2-VPC] The primary IPv4 address. You must specify a value from the IPv4
	// address range of the subnet.
	//
	// Only one private IP address can be designated as primary. You can't specify
	// this option if you've specified the option to designate a private IP address
	// as the primary IP address in a network interface specification. You cannot
	// specify this option if you're launching more than one instance in the request.
	//
	// You cannot specify this option and the network interfaces option in the same
	// request.
	PrivateIPAddress *string `json:"privateIPAddress,omitempty"`
	// The ID of the RAM disk to select. Some kernels require additional drivers
	// at launch. Check the kernel requirements for information about whether you
	// need to specify a RAM disk. To find kernel requirements, go to the Amazon
	// Web Services Resource Center and search for the kernel ID.
	//
	// We recommend that you use PV-GRUB instead of kernels and RAM disks. For more
	// information, see PV-GRUB (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/UserProvidedkernels.html)
	// in the Amazon EC2 User Guide.
	RAMDiskID *string `json:"ramDiskID,omitempty"`
	// The IDs of the security groups. You can create a security group using CreateSecurityGroup
	// (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateSecurityGroup.html).
	//
	// If you specify a network interface, you must specify any security groups
	// as part of the network interface.
	SecurityGroupIDs []*string `json:"securityGroupIDs,omitempty"`
	// [EC2-Classic, default VPC] The names of the security groups. For a nondefault
	// VPC, you must use security group IDs instead.
	//
	// If you specify a network interface, you must specify any security groups
	// as part of the network interface.
	//
	// Default: Amazon EC2 uses the default security group.
	SecurityGroups []*string `json:"securityGroups,omitempty"`
	// [EC2-VPC] The ID of the subnet to launch the instance into.
	//
	// If you specify a network interface, you must specify any subnets as part
	// of the network interface.
	SubnetID *string `json:"subnetID,omitempty"`
	// The tags. The value parameter is required, but if you don't want the tag
	// to have a value, specify the parameter with no value, and we set the value
	// to an empty string.
	Tags []*Tag `json:"tags,omitempty"`
	// The user data to make available to the instance. For more information, see
	// Running commands on your Linux instance at launch (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html)
	// (Linux) and Adding User Data (https://docs.aws.amazon.com/AWSEC2/latest/WindowsGuide/ec2-instance-metadata.html#instancedata-add-user-data)
	// (Windows). If you are using a command line tool, base64-encoding is performed
	// for you, and you can load the text from a file. Otherwise, you must provide
	// base64-encoded text. User data is limited to 16 KB.
	UserData *string `json:"userData,omitempty"`
}

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	// All CRs managed by ACK have a common `Status.ACKResourceMetadata` member
	// that is used to contain resource sync state, account ownership,
	// constructed ARN for the resource
	// +kubebuilder:validation:Optional
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	// All CRS managed by ACK have a common `Status.Conditions` member that
	// contains a collection of `ackv1alpha1.Condition` objects that describe
	// the various terminal states of the CR and its backend AWS service API
	// resource
	// +kubebuilder:validation:Optional
	Conditions []*ackv1alpha1.Condition `json:"conditions"`
	// The AMI launch index, which can be used to find this instance in the launch
	// group.
	// +kubebuilder:validation:Optional
	AMILaunchIndex *int64 `json:"amiLaunchIndex,omitempty"`
	// The architecture of the image.
	// +kubebuilder:validation:Optional
	Architecture *string `json:"architecture,omitempty"`
	// The boot mode of the instance. For more information, see Boot modes (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ami-boot.html)
	// in the Amazon EC2 User Guide.
	// +kubebuilder:validation:Optional
	BootMode *string `json:"bootMode,omitempty"`
	// The ID of the Capacity Reservation.
	// +kubebuilder:validation:Optional
	CapacityReservationID *string `json:"capacityReservationID,omitempty"`
	// The Elastic GPU associated with the instance.
	// +kubebuilder:validation:Optional
	ElasticGPUAssociations []*ElasticGPUAssociation `json:"elasticGPUAssociations,omitempty"`
	// The elastic inference accelerator associated with the instance.
	// +kubebuilder:validation:Optional
	ElasticInferenceAcceleratorAssociations []*ElasticInferenceAcceleratorAssociation `json:"elasticInferenceAcceleratorAssociations,omitempty"`
	// Specifies whether enhanced networking with ENA is enabled.
	// +kubebuilder:validation:Optional
	ENASupport *bool `json:"enaSupport,omitempty"`
	// The hypervisor type of the instance. The value xen is used for both Xen and
	// Nitro hypervisors.
	// +kubebuilder:validation:Optional
	Hypervisor *string `json:"hypervisor,omitempty"`
	// The ID of the instance.
	// +kubebuilder:validation:Optional
	InstanceID *string `json:"instanceID,omitempty"`
	// Indicates whether this is a Spot Instance or a Scheduled Instance.
	// +kubebuilder:validation:Optional
	InstanceLifecycle *string `json:"instanceLifecycle,omitempty"`
	// The time the instance was launched.
	// +kubebuilder:validation:Optional
	LaunchTime *metav1.Time `json:"launchTime,omitempty"`
	// The license configurations for the instance.
	// +kubebuilder:validation:Optional
	Licenses []*LicenseConfiguration `json:"licenses,omitempty"`
	// The Amazon Resource Name (ARN) of the Outpost.
	// +kubebuilder:validation:Optional
	OutpostARN *string `json:"outpostARN,omitempty"`
	// The value is Windows for Windows instances; otherwise blank.
	// +kubebuilder:validation:Optional
	Platform *string `json:"platform,omitempty"`
	// The platform details value for the instance. For more information, see AMI
	// billing information fields (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/billing-info-fields.html)
	// in the Amazon EC2 User Guide.
	// +kubebuilder:validation:Optional
	PlatformDetails *string `json:"platformDetails,omitempty"`
	// (IPv4 only) The private DNS hostname name assigned to the instance. This
	// DNS hostname can only be used inside the Amazon EC2 network. This name is
	// not available until the instance enters the running state.
	//
	// [EC2-VPC] The Amazon-provided DNS server resolves Amazon-provided private
	// DNS hostnames if you've enabled DNS resolution and DNS hostnames in your
	// VPC. If you are not using the Amazon-provided DNS server in your VPC, your
	// custom domain name servers must resolve the hostname as appropriate.
	// +kubebuilder:validation:Optional
	PrivateDNSName *string `json:"privateDNSName,omitempty"`
	// The product codes attached to this instance, if applicable.
	// +kubebuilder:validation:Optional
	ProductCodes []*ProductCode `json:"productCodes,omitempty"`
	// (IPv4 only) The public DNS name assigned to the instance. This name is not
	// available until the instance enters the running state. For EC2-VPC, this
	// name is only available if you've enabled DNS hostnames for your VPC.
	// +kubebuilder:validation:Optional
	PublicDNSName *string `json:"publicDNSName,omitempty"`
	// The public IPv4 address, or the Carrier IP address assigned to the instance,
	// if applicable.
	//
	// A Carrier IP address only applies to an instance launched in a subnet associated
	// with a Wavelength Zone.
	// +kubebuilder:validation:Optional
	PublicIPAddress *string `json:"publicIPAddress,omitempty"`
	// The device name of the root device volume (for example, /dev/sda1).
	// +kubebuilder:validation:Optional
	RootDeviceName *string `json:"rootDeviceName,omitempty"`
	// The root device type used by the AMI. The AMI can use an EBS volume or an
	// instance store volume.
	// +kubebuilder:validation:Optional
	RootDeviceType *string `json:"rootDeviceType,omitempty"`
	// Indicates whether source/destination checking is enabled.
	// +kubebuilder:validation:Optional
	SourceDestCheck *bool `json:"sourceDestCheck,omitempty"`
	// If the request is a Spot Instance request, the ID of the request.
	// +kubebuilder:validation:Optional
	SpotInstanceRequestID *string `json:"spotInstanceRequestID,omitempty"`
	// Specifies whether enhanced networking with the Intel 82599 Virtual Function
	// interface is enabled.
	// +kubebuilder:validation:Optional
	SRIOVNetSupport *string `json:"sriovNetSupport,omitempty"`
	// The current state of the instance.
	// +kubebuilder:validation:Optional
	State *InstanceState `json:"state,omitempty"`
	// The reason for the most recent state transition.
	// +kubebuilder:validation:Optional
	StateReason *StateReason `json:"stateReason,omitempty"`
	// The reason for the most recent state transition. This might be an empty string.
	// +kubebuilder:validation:Optional
	StateTransitionReason *string `json:"stateTransitionReason,omitempty"`
	// The usage operation value for the instance. For more information, see AMI
	// billing information fields (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/billing-info-fields.html)
	// in the Amazon EC2 User Guide.
	// +kubebuilder:validation:Optional
	UsageOperation *string `json:"usageOperation,omitempty"`
	// The time that the usage operation was last updated.
	// +kubebuilder:validation:Optional
	UsageOperationUpdateTime *metav1.Time `json:"usageOperationUpdateTime,omitempty"`
	// The virtualization type of the instance.
	// +kubebuilder:validation:Optional
	VirtualizationType *string `json:"virtualizationType,omitempty"`
	// [EC2-VPC] The ID of the VPC in which the instance is running.
	// +kubebuilder:validation:Optional
	VPCID *string `json:"vpcID,omitempty"`
}

// Instance is the Schema for the Instances API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstanceSpec   `json:"spec,omitempty"`
	Status            InstanceStatus `json:"status,omitempty"`
}

// InstanceList contains a list of Instance
// +kubebuilder:object:root=true
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
