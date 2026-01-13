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

package fleet

import (
	"context"
	"fmt"
	"math"
	"strconv"

	// svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
	// ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"

	// ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	// corev1 "k8s.io/api/core/v1"
)

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcPeeringConnectionInput.TagSpecification
// and ensures the ResourceType is always set to 'fleet'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateFleetInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.Key = desiredTag.Key
				tag.Value = desiredTag.Value
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.ResourceType = "fleet"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

var syncTags = tags.Sync

var computeTagsDelta = tags.ComputeTagsDelta

// customUpdateFleet patches the supplied resource in the backend AWS service API and
// returns a new resource with updated fields.
func (rm *resourceManager) customUpdateFleet(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateFleet")
	defer func() {
		exit(err)
	}()
	if delta.DifferentAt("Spec.Tags") {
		if err := syncTags(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.FleetID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	if !delta.DifferentExcept("Spec.Tags") {
		return desired, nil
	}

	if delta.DifferentAt("Spec.TargetCapacitySpecification.DefaultTargetCapacityType") {
		// Throw a Terminal Error if the field is modified
		if latest.ko.Spec.TargetCapacitySpecification.DefaultTargetCapacityType != desired.ko.Spec.TargetCapacitySpecification.DefaultTargetCapacityType {
			msg := "Currently we don’t support changing Default Target Capacity Type"
			return nil, ackerr.NewTerminalError(fmt.Errorf("%s", msg))
		}
	}

	input, err := rm.customUpdateRequestPayload(ctx, desired, latest, delta)
	if err != nil {
		return nil, err
	}

	for _, config := range input.LaunchTemplateConfigs {
		if config.LaunchTemplateSpecification != nil {

			// Ensure Launch Template Version is int and not $Latest/$Default
			// Preventing those values as as those strings are updated in the backend with the ints they represent
			// This confuses the reconciliation as the aws state falls out of sync with the CRD
			// If we stick to int strings, all works smoothly

			_, err := strconv.Atoi(*config.LaunchTemplateSpecification.Version)
			if err != nil {
				msg := "Only int values are supported for Launch Template Version in EC2 fleet spec"
				return nil, ackerr.NewTerminalError(fmt.Errorf("%s", msg))
			}
		}
	}

	var resp *svcsdk.ModifyFleetOutput
	_ = resp
	resp, err = rm.sdkapi.ModifyFleet(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyFleet", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	// Modify Fleet doesn't return an updated Fleet object, so we need to set the state to "modifying" to reflect that the update is in progress
	ko.Status.FleetState = aws.String("modifying")

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// customUpdateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Update API call for the resource
func (rm *resourceManager) customUpdateRequestPayload(
	ctx context.Context,
	r *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*svcsdk.ModifyFleetInput, error) {
	res := &svcsdk.ModifyFleetInput{}

	if r.ko.Spec.Context != nil {
		res.Context = r.ko.Spec.Context
	}
	if r.ko.Spec.ExcessCapacityTerminationPolicy != nil {
		res.ExcessCapacityTerminationPolicy = svcsdktypes.FleetExcessCapacityTerminationPolicy(*r.ko.Spec.ExcessCapacityTerminationPolicy)
	}
	if r.ko.Status.FleetID != nil {
		res.FleetId = r.ko.Status.FleetID
	}
	if r.ko.Spec.LaunchTemplateConfigs != nil {
		f4 := []svcsdktypes.FleetLaunchTemplateConfigRequest{}
		for _, f4iter := range r.ko.Spec.LaunchTemplateConfigs {
			f4elem := &svcsdktypes.FleetLaunchTemplateConfigRequest{}
			if f4iter.LaunchTemplateSpecification != nil {
				f4elemf0 := &svcsdktypes.FleetLaunchTemplateSpecificationRequest{}
				if f4iter.LaunchTemplateSpecification.LaunchTemplateID != nil {
					f4elemf0.LaunchTemplateId = f4iter.LaunchTemplateSpecification.LaunchTemplateID
				}
				if f4iter.LaunchTemplateSpecification.LaunchTemplateName != nil {
					f4elemf0.LaunchTemplateName = f4iter.LaunchTemplateSpecification.LaunchTemplateName
				}
				if f4iter.LaunchTemplateSpecification.Version != nil {
					f4elemf0.Version = f4iter.LaunchTemplateSpecification.Version
				}
				f4elem.LaunchTemplateSpecification = f4elemf0
			}
			if f4iter.Overrides != nil {
				f4elemf1 := []svcsdktypes.FleetLaunchTemplateOverridesRequest{}
				for _, f4elemf1iter := range f4iter.Overrides {
					f4elemf1elem := &svcsdktypes.FleetLaunchTemplateOverridesRequest{}
					if f4elemf1iter.AvailabilityZone != nil {
						f4elemf1elem.AvailabilityZone = f4elemf1iter.AvailabilityZone
					}
					if f4elemf1iter.ImageID != nil {
						f4elemf1elem.ImageId = f4elemf1iter.ImageID
					}
					if f4elemf1iter.InstanceRequirements != nil {
						f4elemf1elemf2 := &svcsdktypes.InstanceRequirementsRequest{}
						if f4elemf1iter.InstanceRequirements.AcceleratorCount != nil {
							f4elemf1elemf2f0 := &svcsdktypes.AcceleratorCountRequest{}
							if f4elemf1iter.InstanceRequirements.AcceleratorCount.Max != nil {
								maxCopy0 := *f4elemf1iter.InstanceRequirements.AcceleratorCount.Max
								if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Max is of type int32")
								}
								maxCopy := int32(maxCopy0)
								f4elemf1elemf2f0.Max = &maxCopy
							}
							if f4elemf1iter.InstanceRequirements.AcceleratorCount.Min != nil {
								minCopy0 := *f4elemf1iter.InstanceRequirements.AcceleratorCount.Min
								if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Min is of type int32")
								}
								minCopy := int32(minCopy0)
								f4elemf1elemf2f0.Min = &minCopy
							}
							f4elemf1elemf2.AcceleratorCount = f4elemf1elemf2f0
						}
						if f4elemf1iter.InstanceRequirements.AcceleratorManufacturers != nil {
							f4elemf1elemf2f1 := []svcsdktypes.AcceleratorManufacturer{}
							for _, f4elemf1elemf2f1iter := range f4elemf1iter.InstanceRequirements.AcceleratorManufacturers {
								var f4elemf1elemf2f1elem string
								f4elemf1elemf2f1elem = string(*f4elemf1elemf2f1iter)
								f4elemf1elemf2f1 = append(f4elemf1elemf2f1, svcsdktypes.AcceleratorManufacturer(f4elemf1elemf2f1elem))
							}
							f4elemf1elemf2.AcceleratorManufacturers = f4elemf1elemf2f1
						}
						if f4elemf1iter.InstanceRequirements.AcceleratorNames != nil {
							f4elemf1elemf2f2 := []svcsdktypes.AcceleratorName{}
							for _, f4elemf1elemf2f2iter := range f4elemf1iter.InstanceRequirements.AcceleratorNames {
								var f4elemf1elemf2f2elem string
								f4elemf1elemf2f2elem = string(*f4elemf1elemf2f2iter)
								f4elemf1elemf2f2 = append(f4elemf1elemf2f2, svcsdktypes.AcceleratorName(f4elemf1elemf2f2elem))
							}
							f4elemf1elemf2.AcceleratorNames = f4elemf1elemf2f2
						}
						if f4elemf1iter.InstanceRequirements.AcceleratorTotalMemoryMiB != nil {
							f4elemf1elemf2f3 := &svcsdktypes.AcceleratorTotalMemoryMiBRequest{}
							if f4elemf1iter.InstanceRequirements.AcceleratorTotalMemoryMiB.Max != nil {
								maxCopy0 := *f4elemf1iter.InstanceRequirements.AcceleratorTotalMemoryMiB.Max
								if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Max is of type int32")
								}
								maxCopy := int32(maxCopy0)
								f4elemf1elemf2f3.Max = &maxCopy
							}
							if f4elemf1iter.InstanceRequirements.AcceleratorTotalMemoryMiB.Min != nil {
								minCopy0 := *f4elemf1iter.InstanceRequirements.AcceleratorTotalMemoryMiB.Min
								if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Min is of type int32")
								}
								minCopy := int32(minCopy0)
								f4elemf1elemf2f3.Min = &minCopy
							}
							f4elemf1elemf2.AcceleratorTotalMemoryMiB = f4elemf1elemf2f3
						}
						if f4elemf1iter.InstanceRequirements.AcceleratorTypes != nil {
							f4elemf1elemf2f4 := []svcsdktypes.AcceleratorType{}
							for _, f4elemf1elemf2f4iter := range f4elemf1iter.InstanceRequirements.AcceleratorTypes {
								var f4elemf1elemf2f4elem string
								f4elemf1elemf2f4elem = string(*f4elemf1elemf2f4iter)
								f4elemf1elemf2f4 = append(f4elemf1elemf2f4, svcsdktypes.AcceleratorType(f4elemf1elemf2f4elem))
							}
							f4elemf1elemf2.AcceleratorTypes = f4elemf1elemf2f4
						}
						if f4elemf1iter.InstanceRequirements.AllowedInstanceTypes != nil {
							f4elemf1elemf2.AllowedInstanceTypes = aws.ToStringSlice(f4elemf1iter.InstanceRequirements.AllowedInstanceTypes)
						}
						if f4elemf1iter.InstanceRequirements.BareMetal != nil {
							f4elemf1elemf2.BareMetal = svcsdktypes.BareMetal(*f4elemf1iter.InstanceRequirements.BareMetal)
						}
						if f4elemf1iter.InstanceRequirements.BaselineEBSBandwidthMbps != nil {
							f4elemf1elemf2f7 := &svcsdktypes.BaselineEbsBandwidthMbpsRequest{}
							if f4elemf1iter.InstanceRequirements.BaselineEBSBandwidthMbps.Max != nil {
								maxCopy0 := *f4elemf1iter.InstanceRequirements.BaselineEBSBandwidthMbps.Max
								if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Max is of type int32")
								}
								maxCopy := int32(maxCopy0)
								f4elemf1elemf2f7.Max = &maxCopy
							}
							if f4elemf1iter.InstanceRequirements.BaselineEBSBandwidthMbps.Min != nil {
								minCopy0 := *f4elemf1iter.InstanceRequirements.BaselineEBSBandwidthMbps.Min
								if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Min is of type int32")
								}
								minCopy := int32(minCopy0)
								f4elemf1elemf2f7.Min = &minCopy
							}
							f4elemf1elemf2.BaselineEbsBandwidthMbps = f4elemf1elemf2f7
						}
						if f4elemf1iter.InstanceRequirements.BaselinePerformanceFactors != nil {
							f4elemf1elemf2f8 := &svcsdktypes.BaselinePerformanceFactorsRequest{}
							if f4elemf1iter.InstanceRequirements.BaselinePerformanceFactors.CPU != nil {
								f4elemf1elemf2f8f0 := &svcsdktypes.CpuPerformanceFactorRequest{}
								if f4elemf1iter.InstanceRequirements.BaselinePerformanceFactors.CPU.References != nil {
									f4elemf1elemf2f8f0f0 := []svcsdktypes.PerformanceFactorReferenceRequest{}
									for _, f4elemf1elemf2f8f0f0iter := range f4elemf1iter.InstanceRequirements.BaselinePerformanceFactors.CPU.References {
										f4elemf1elemf2f8f0f0elem := &svcsdktypes.PerformanceFactorReferenceRequest{}
										if f4elemf1elemf2f8f0f0iter.InstanceFamily != nil {
											f4elemf1elemf2f8f0f0elem.InstanceFamily = f4elemf1elemf2f8f0f0iter.InstanceFamily
										}
										f4elemf1elemf2f8f0f0 = append(f4elemf1elemf2f8f0f0, *f4elemf1elemf2f8f0f0elem)
									}
									f4elemf1elemf2f8f0.References = f4elemf1elemf2f8f0f0
								}
								f4elemf1elemf2f8.Cpu = f4elemf1elemf2f8f0
							}
							f4elemf1elemf2.BaselinePerformanceFactors = f4elemf1elemf2f8
						}
						if f4elemf1iter.InstanceRequirements.BurstablePerformance != nil {
							f4elemf1elemf2.BurstablePerformance = svcsdktypes.BurstablePerformance(*f4elemf1iter.InstanceRequirements.BurstablePerformance)
						}
						if f4elemf1iter.InstanceRequirements.CPUManufacturers != nil {
							f4elemf1elemf2f10 := []svcsdktypes.CpuManufacturer{}
							for _, f4elemf1elemf2f10iter := range f4elemf1iter.InstanceRequirements.CPUManufacturers {
								var f4elemf1elemf2f10elem string
								f4elemf1elemf2f10elem = string(*f4elemf1elemf2f10iter)
								f4elemf1elemf2f10 = append(f4elemf1elemf2f10, svcsdktypes.CpuManufacturer(f4elemf1elemf2f10elem))
							}
							f4elemf1elemf2.CpuManufacturers = f4elemf1elemf2f10
						}
						if f4elemf1iter.InstanceRequirements.ExcludedInstanceTypes != nil {
							f4elemf1elemf2.ExcludedInstanceTypes = aws.ToStringSlice(f4elemf1iter.InstanceRequirements.ExcludedInstanceTypes)
						}
						if f4elemf1iter.InstanceRequirements.InstanceGenerations != nil {
							f4elemf1elemf2f12 := []svcsdktypes.InstanceGeneration{}
							for _, f4elemf1elemf2f12iter := range f4elemf1iter.InstanceRequirements.InstanceGenerations {
								var f4elemf1elemf2f12elem string
								f4elemf1elemf2f12elem = string(*f4elemf1elemf2f12iter)
								f4elemf1elemf2f12 = append(f4elemf1elemf2f12, svcsdktypes.InstanceGeneration(f4elemf1elemf2f12elem))
							}
							f4elemf1elemf2.InstanceGenerations = f4elemf1elemf2f12
						}
						if f4elemf1iter.InstanceRequirements.LocalStorage != nil {
							f4elemf1elemf2.LocalStorage = svcsdktypes.LocalStorage(*f4elemf1iter.InstanceRequirements.LocalStorage)
						}
						if f4elemf1iter.InstanceRequirements.LocalStorageTypes != nil {
							f4elemf1elemf2f14 := []svcsdktypes.LocalStorageType{}
							for _, f4elemf1elemf2f14iter := range f4elemf1iter.InstanceRequirements.LocalStorageTypes {
								var f4elemf1elemf2f14elem string
								f4elemf1elemf2f14elem = string(*f4elemf1elemf2f14iter)
								f4elemf1elemf2f14 = append(f4elemf1elemf2f14, svcsdktypes.LocalStorageType(f4elemf1elemf2f14elem))
							}
							f4elemf1elemf2.LocalStorageTypes = f4elemf1elemf2f14
						}
						if f4elemf1iter.InstanceRequirements.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice != nil {
							maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0 := *f4elemf1iter.InstanceRequirements.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice
							if maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0 > math.MaxInt32 || maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0 < math.MinInt32 {
								return nil, fmt.Errorf("error: field MaxSpotPriceAsPercentageOfOptimalOnDemandPrice is of type int32")
							}
							maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy := int32(maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0)
							f4elemf1elemf2.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice = &maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy
						}
						if f4elemf1iter.InstanceRequirements.MemoryGiBPerVCPU != nil {
							f4elemf1elemf2f16 := &svcsdktypes.MemoryGiBPerVCpuRequest{}
							if f4elemf1iter.InstanceRequirements.MemoryGiBPerVCPU.Max != nil {
								f4elemf1elemf2f16.Max = f4elemf1iter.InstanceRequirements.MemoryGiBPerVCPU.Max
							}
							if f4elemf1iter.InstanceRequirements.MemoryGiBPerVCPU.Min != nil {
								f4elemf1elemf2f16.Min = f4elemf1iter.InstanceRequirements.MemoryGiBPerVCPU.Min
							}
							f4elemf1elemf2.MemoryGiBPerVCpu = f4elemf1elemf2f16
						}
						if f4elemf1iter.InstanceRequirements.MemoryMiB != nil {
							f4elemf1elemf2f17 := &svcsdktypes.MemoryMiBRequest{}
							if f4elemf1iter.InstanceRequirements.MemoryMiB.Max != nil {
								maxCopy0 := *f4elemf1iter.InstanceRequirements.MemoryMiB.Max
								if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Max is of type int32")
								}
								maxCopy := int32(maxCopy0)
								f4elemf1elemf2f17.Max = &maxCopy
							}
							if f4elemf1iter.InstanceRequirements.MemoryMiB.Min != nil {
								minCopy0 := *f4elemf1iter.InstanceRequirements.MemoryMiB.Min
								if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Min is of type int32")
								}
								minCopy := int32(minCopy0)
								f4elemf1elemf2f17.Min = &minCopy
							}
							f4elemf1elemf2.MemoryMiB = f4elemf1elemf2f17
						}
						if f4elemf1iter.InstanceRequirements.NetworkBandwidthGbps != nil {
							f4elemf1elemf2f18 := &svcsdktypes.NetworkBandwidthGbpsRequest{}
							if f4elemf1iter.InstanceRequirements.NetworkBandwidthGbps.Max != nil {
								f4elemf1elemf2f18.Max = f4elemf1iter.InstanceRequirements.NetworkBandwidthGbps.Max
							}
							if f4elemf1iter.InstanceRequirements.NetworkBandwidthGbps.Min != nil {
								f4elemf1elemf2f18.Min = f4elemf1iter.InstanceRequirements.NetworkBandwidthGbps.Min
							}
							f4elemf1elemf2.NetworkBandwidthGbps = f4elemf1elemf2f18
						}
						if f4elemf1iter.InstanceRequirements.NetworkInterfaceCount != nil {
							f4elemf1elemf2f19 := &svcsdktypes.NetworkInterfaceCountRequest{}
							if f4elemf1iter.InstanceRequirements.NetworkInterfaceCount.Max != nil {
								maxCopy0 := *f4elemf1iter.InstanceRequirements.NetworkInterfaceCount.Max
								if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Max is of type int32")
								}
								maxCopy := int32(maxCopy0)
								f4elemf1elemf2f19.Max = &maxCopy
							}
							if f4elemf1iter.InstanceRequirements.NetworkInterfaceCount.Min != nil {
								minCopy0 := *f4elemf1iter.InstanceRequirements.NetworkInterfaceCount.Min
								if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Min is of type int32")
								}
								minCopy := int32(minCopy0)
								f4elemf1elemf2f19.Min = &minCopy
							}
							f4elemf1elemf2.NetworkInterfaceCount = f4elemf1elemf2f19
						}
						if f4elemf1iter.InstanceRequirements.OnDemandMaxPricePercentageOverLowestPrice != nil {
							onDemandMaxPricePercentageOverLowestPriceCopy0 := *f4elemf1iter.InstanceRequirements.OnDemandMaxPricePercentageOverLowestPrice
							if onDemandMaxPricePercentageOverLowestPriceCopy0 > math.MaxInt32 || onDemandMaxPricePercentageOverLowestPriceCopy0 < math.MinInt32 {
								return nil, fmt.Errorf("error: field OnDemandMaxPricePercentageOverLowestPrice is of type int32")
							}
							onDemandMaxPricePercentageOverLowestPriceCopy := int32(onDemandMaxPricePercentageOverLowestPriceCopy0)
							f4elemf1elemf2.OnDemandMaxPricePercentageOverLowestPrice = &onDemandMaxPricePercentageOverLowestPriceCopy
						}
						if f4elemf1iter.InstanceRequirements.RequireHibernateSupport != nil {
							f4elemf1elemf2.RequireHibernateSupport = f4elemf1iter.InstanceRequirements.RequireHibernateSupport
						}
						if f4elemf1iter.InstanceRequirements.SpotMaxPricePercentageOverLowestPrice != nil {
							spotMaxPricePercentageOverLowestPriceCopy0 := *f4elemf1iter.InstanceRequirements.SpotMaxPricePercentageOverLowestPrice
							if spotMaxPricePercentageOverLowestPriceCopy0 > math.MaxInt32 || spotMaxPricePercentageOverLowestPriceCopy0 < math.MinInt32 {
								return nil, fmt.Errorf("error: field SpotMaxPricePercentageOverLowestPrice is of type int32")
							}
							spotMaxPricePercentageOverLowestPriceCopy := int32(spotMaxPricePercentageOverLowestPriceCopy0)
							f4elemf1elemf2.SpotMaxPricePercentageOverLowestPrice = &spotMaxPricePercentageOverLowestPriceCopy
						}
						if f4elemf1iter.InstanceRequirements.TotalLocalStorageGB != nil {
							f4elemf1elemf2f23 := &svcsdktypes.TotalLocalStorageGBRequest{}
							if f4elemf1iter.InstanceRequirements.TotalLocalStorageGB.Max != nil {
								f4elemf1elemf2f23.Max = f4elemf1iter.InstanceRequirements.TotalLocalStorageGB.Max
							}
							if f4elemf1iter.InstanceRequirements.TotalLocalStorageGB.Min != nil {
								f4elemf1elemf2f23.Min = f4elemf1iter.InstanceRequirements.TotalLocalStorageGB.Min
							}
							f4elemf1elemf2.TotalLocalStorageGB = f4elemf1elemf2f23
						}
						if f4elemf1iter.InstanceRequirements.VCPUCount != nil {
							f4elemf1elemf2f24 := &svcsdktypes.VCpuCountRangeRequest{}
							if f4elemf1iter.InstanceRequirements.VCPUCount.Max != nil {
								maxCopy0 := *f4elemf1iter.InstanceRequirements.VCPUCount.Max
								if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Max is of type int32")
								}
								maxCopy := int32(maxCopy0)
								f4elemf1elemf2f24.Max = &maxCopy
							}
							if f4elemf1iter.InstanceRequirements.VCPUCount.Min != nil {
								minCopy0 := *f4elemf1iter.InstanceRequirements.VCPUCount.Min
								if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
									return nil, fmt.Errorf("error: field Min is of type int32")
								}
								minCopy := int32(minCopy0)
								f4elemf1elemf2f24.Min = &minCopy
							}
							f4elemf1elemf2.VCpuCount = f4elemf1elemf2f24
						}
						f4elemf1elem.InstanceRequirements = f4elemf1elemf2
					}
					if f4elemf1iter.InstanceType != nil {
						f4elemf1elem.InstanceType = svcsdktypes.InstanceType(*f4elemf1iter.InstanceType)
					}
					if f4elemf1iter.MaxPrice != nil {
						f4elemf1elem.MaxPrice = f4elemf1iter.MaxPrice
					}
					if f4elemf1iter.Placement != nil {
						f4elemf1elemf5 := &svcsdktypes.Placement{}
						if f4elemf1iter.Placement.Affinity != nil {
							f4elemf1elemf5.Affinity = f4elemf1iter.Placement.Affinity
						}
						if f4elemf1iter.Placement.AvailabilityZone != nil {
							f4elemf1elemf5.AvailabilityZone = f4elemf1iter.Placement.AvailabilityZone
						}
						if f4elemf1iter.Placement.GroupName != nil {
							f4elemf1elemf5.GroupName = f4elemf1iter.Placement.GroupName
						}
						if f4elemf1iter.Placement.HostID != nil {
							f4elemf1elemf5.HostId = f4elemf1iter.Placement.HostID
						}
						if f4elemf1iter.Placement.HostResourceGroupARN != nil {
							f4elemf1elemf5.HostResourceGroupArn = f4elemf1iter.Placement.HostResourceGroupARN
						}
						if f4elemf1iter.Placement.PartitionNumber != nil {
							partitionNumberCopy0 := *f4elemf1iter.Placement.PartitionNumber
							if partitionNumberCopy0 > math.MaxInt32 || partitionNumberCopy0 < math.MinInt32 {
								return nil, fmt.Errorf("error: field PartitionNumber is of type int32")
							}
							partitionNumberCopy := int32(partitionNumberCopy0)
							f4elemf1elemf5.PartitionNumber = &partitionNumberCopy
						}
						if f4elemf1iter.Placement.SpreadDomain != nil {
							f4elemf1elemf5.SpreadDomain = f4elemf1iter.Placement.SpreadDomain
						}
						if f4elemf1iter.Placement.Tenancy != nil {
							f4elemf1elemf5.Tenancy = svcsdktypes.Tenancy(*f4elemf1iter.Placement.Tenancy)
						}
						f4elemf1elem.Placement = f4elemf1elemf5
					}
					if f4elemf1iter.Priority != nil {
						f4elemf1elem.Priority = f4elemf1iter.Priority
					}
					if f4elemf1iter.SubnetID != nil {
						f4elemf1elem.SubnetId = f4elemf1iter.SubnetID
					}
					if f4elemf1iter.WeightedCapacity != nil {
						f4elemf1elem.WeightedCapacity = f4elemf1iter.WeightedCapacity
					}
					f4elemf1 = append(f4elemf1, *f4elemf1elem)
				}
				f4elem.Overrides = f4elemf1
			}
			f4 = append(f4, *f4elem)
		}
		res.LaunchTemplateConfigs = f4
	}
	if r.ko.Spec.TargetCapacitySpecification != nil {
		f5 := &svcsdktypes.TargetCapacitySpecificationRequest{}
		if r.ko.Spec.TargetCapacitySpecification.OnDemandTargetCapacity != nil {
			onDemandTargetCapacityCopy0 := *r.ko.Spec.TargetCapacitySpecification.OnDemandTargetCapacity
			if onDemandTargetCapacityCopy0 > math.MaxInt32 || onDemandTargetCapacityCopy0 < math.MinInt32 {
				return nil, fmt.Errorf("error: field OnDemandTargetCapacity is of type int32")
			}
			onDemandTargetCapacityCopy := int32(onDemandTargetCapacityCopy0)
			f5.OnDemandTargetCapacity = &onDemandTargetCapacityCopy
		}
		if r.ko.Spec.TargetCapacitySpecification.SpotTargetCapacity != nil {
			spotTargetCapacityCopy0 := *r.ko.Spec.TargetCapacitySpecification.SpotTargetCapacity
			if spotTargetCapacityCopy0 > math.MaxInt32 || spotTargetCapacityCopy0 < math.MinInt32 {
				return nil, fmt.Errorf("error: field SpotTargetCapacity is of type int32")
			}
			spotTargetCapacityCopy := int32(spotTargetCapacityCopy0)
			f5.SpotTargetCapacity = &spotTargetCapacityCopy
		}
		if r.ko.Spec.TargetCapacitySpecification.TargetCapacityUnitType != nil {
			f5.TargetCapacityUnitType = svcsdktypes.TargetCapacityUnitType(*r.ko.Spec.TargetCapacitySpecification.TargetCapacityUnitType)
		}
		if r.ko.Spec.TargetCapacitySpecification.TotalTargetCapacity != nil {
			totalTargetCapacityCopy0 := *r.ko.Spec.TargetCapacitySpecification.TotalTargetCapacity
			if totalTargetCapacityCopy0 > math.MaxInt32 || totalTargetCapacityCopy0 < math.MinInt32 {
				return nil, fmt.Errorf("error: field TotalTargetCapacity is of type int32")
			}
			totalTargetCapacityCopy := int32(totalTargetCapacityCopy0)
			f5.TotalTargetCapacity = &totalTargetCapacityCopy
		}
		res.TargetCapacitySpecification = f5
	}

	return res, nil
}

// addIDToDeleteRequest adds resource's Fleet ID to DeleteRequest.
// Return error to indicate to callers that the resource is not yet created.
func addIDToDeleteRequest(r *resource,
	input *svcsdk.DeleteFleetsInput) error {
	if r.ko.Status.FleetID == nil {
		return fmt.Errorf("unable to extract FleetID from resource")
	}
	input.FleetIds = []string{*r.ko.Status.FleetID}
	return nil
}
