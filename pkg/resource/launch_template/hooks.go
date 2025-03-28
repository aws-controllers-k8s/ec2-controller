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

package launch_template

import (
	"context"
	"fmt"

	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) findLaunchTemplateVersion(
	ctx context.Context,
	r *resource,
	ko *svcapitypes.LaunchTemplate,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.findLaunchTemplateVersion")
	defer func() {
		exit(err)
	}()

	input, err := newListLaunchTemplateVersionRequestPayload(r)

	if err != nil {
		return err
	}

	var resp *svcsdk.DescribeLaunchTemplateVersionsOutput
	resp, err = rm.sdkapi.DescribeLaunchTemplateVersions(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeLaunchTemplateVersions", err)
	if err != nil {
		return err
	}

	for _, elem := range resp.LaunchTemplateVersions {
		if elem.CreateTime != nil {
			ko.Status.CreateTime = &metav1.Time{*elem.CreateTime}
		} else {
			ko.Status.CreateTime = nil
		}
		if elem.CreatedBy != nil {
			ko.Status.CreatedBy = elem.CreatedBy
		} else {
			ko.Status.CreatedBy = nil
		}
		if elem.LaunchTemplateData != nil {
			f3 := &svcapitypes.RequestLaunchTemplateData{}
			if elem.LaunchTemplateData.BlockDeviceMappings != nil {
				f3f0 := []*svcapitypes.LaunchTemplateBlockDeviceMappingRequest{}
				for _, f3f0iter := range elem.LaunchTemplateData.BlockDeviceMappings {
					f3f0elem := &svcapitypes.LaunchTemplateBlockDeviceMappingRequest{}
					if f3f0iter.DeviceName != nil {
						f3f0elem.DeviceName = f3f0iter.DeviceName
					}
					if f3f0iter.Ebs != nil {
						f3f0elemf1 := &svcapitypes.LaunchTemplateEBSBlockDeviceRequest{}
						if f3f0iter.Ebs.DeleteOnTermination != nil {
							f3f0elemf1.DeleteOnTermination = f3f0iter.Ebs.DeleteOnTermination
						}
						if f3f0iter.Ebs.Encrypted != nil {
							f3f0elemf1.Encrypted = f3f0iter.Ebs.Encrypted
						}
						if f3f0iter.Ebs.Iops != nil {
							iopsCopy := int64(*f3f0iter.Ebs.Iops)
							f3f0elemf1.IOPS = &iopsCopy
						}
						if f3f0iter.Ebs.KmsKeyId != nil {
							f3f0elemf1.KMSKeyID = f3f0iter.Ebs.KmsKeyId
						}
						if f3f0iter.Ebs.SnapshotId != nil {
							f3f0elemf1.SnapshotID = f3f0iter.Ebs.SnapshotId
						}
						if f3f0iter.Ebs.Throughput != nil {
							throughputCopy := int64(*f3f0iter.Ebs.Throughput)
							f3f0elemf1.Throughput = &throughputCopy
						}
						if f3f0iter.Ebs.VolumeSize != nil {
							volumeSizeCopy := int64(*f3f0iter.Ebs.VolumeSize)
							f3f0elemf1.VolumeSize = &volumeSizeCopy
						}
						if f3f0iter.Ebs.VolumeType != "" {
							f3f0elemf1.VolumeType = aws.String(string(f3f0iter.Ebs.VolumeType))
						}
						f3f0elem.EBS = f3f0elemf1
					}
					if f3f0iter.NoDevice != nil {
						f3f0elem.NoDevice = f3f0iter.NoDevice
					}
					if f3f0iter.VirtualName != nil {
						f3f0elem.VirtualName = f3f0iter.VirtualName
					}
					f3f0 = append(f3f0, f3f0elem)
				}
				f3.BlockDeviceMappings = f3f0
			}
			if elem.LaunchTemplateData.CapacityReservationSpecification != nil {
				f3f1 := &svcapitypes.LaunchTemplateCapacityReservationSpecificationRequest{}
				if elem.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationPreference != "" {
					f3f1.CapacityReservationPreference = aws.String(string(elem.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationPreference))
				}
				if elem.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget != nil {
					f3f1f1 := &svcapitypes.CapacityReservationTarget{}
					if elem.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationId != nil {
						f3f1f1.CapacityReservationID = elem.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationId
					}
					if elem.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupArn != nil {
						f3f1f1.CapacityReservationResourceGroupARN = elem.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupArn
					}
					f3f1.CapacityReservationTarget = f3f1f1
				}
				f3.CapacityReservationSpecification = f3f1
			}
			if elem.LaunchTemplateData.CpuOptions != nil {
				f3f2 := &svcapitypes.LaunchTemplateCPUOptionsRequest{}
				if elem.LaunchTemplateData.CpuOptions.AmdSevSnp != "" {
					f3f2.AmdSevSnp = aws.String(string(elem.LaunchTemplateData.CpuOptions.AmdSevSnp))
				}
				if elem.LaunchTemplateData.CpuOptions.CoreCount != nil {
					coreCountCopy := int64(*elem.LaunchTemplateData.CpuOptions.CoreCount)
					f3f2.CoreCount = &coreCountCopy
				}
				if elem.LaunchTemplateData.CpuOptions.ThreadsPerCore != nil {
					threadsPerCoreCopy := int64(*elem.LaunchTemplateData.CpuOptions.ThreadsPerCore)
					f3f2.ThreadsPerCore = &threadsPerCoreCopy
				}
				f3.CPUOptions = f3f2
			}
			if elem.LaunchTemplateData.CreditSpecification != nil {
				f3f3 := &svcapitypes.CreditSpecificationRequest{}
				if elem.LaunchTemplateData.CreditSpecification.CpuCredits != nil {
					f3f3.CPUCredits = elem.LaunchTemplateData.CreditSpecification.CpuCredits
				}
				f3.CreditSpecification = f3f3
			}
			if elem.LaunchTemplateData.DisableApiStop != nil {
				f3.DisableAPIStop = elem.LaunchTemplateData.DisableApiStop
			}
			if elem.LaunchTemplateData.DisableApiTermination != nil {
				f3.DisableAPITermination = elem.LaunchTemplateData.DisableApiTermination
			}
			if elem.LaunchTemplateData.EbsOptimized != nil {
				f3.EBSOptimized = elem.LaunchTemplateData.EbsOptimized
			}
			if elem.LaunchTemplateData.ElasticGpuSpecifications != nil {
				f3f7 := []*svcapitypes.ElasticGPUSpecification{}
				for _, f3f7iter := range elem.LaunchTemplateData.ElasticGpuSpecifications {
					f3f7elem := &svcapitypes.ElasticGPUSpecification{}
					if f3f7iter.Type != nil {
						f3f7elem.Type = f3f7iter.Type
					}
					f3f7 = append(f3f7, f3f7elem)
				}
				f3.ElasticGPUSpecifications = f3f7
			}
			if elem.LaunchTemplateData.ElasticInferenceAccelerators != nil {
				f3f8 := []*svcapitypes.LaunchTemplateElasticInferenceAccelerator{}
				for _, f3f8iter := range elem.LaunchTemplateData.ElasticInferenceAccelerators {
					f3f8elem := &svcapitypes.LaunchTemplateElasticInferenceAccelerator{}
					if f3f8iter.Count != nil {
						countCopy := int64(*f3f8iter.Count)
						f3f8elem.Count = &countCopy
					}
					if f3f8iter.Type != nil {
						f3f8elem.Type = f3f8iter.Type
					}
					f3f8 = append(f3f8, f3f8elem)
				}
				f3.ElasticInferenceAccelerators = f3f8
			}
			if elem.LaunchTemplateData.EnclaveOptions != nil {
				f3f9 := &svcapitypes.LaunchTemplateEnclaveOptionsRequest{}
				if elem.LaunchTemplateData.EnclaveOptions.Enabled != nil {
					f3f9.Enabled = elem.LaunchTemplateData.EnclaveOptions.Enabled
				}
				f3.EnclaveOptions = f3f9
			}
			if elem.LaunchTemplateData.HibernationOptions != nil {
				f3f10 := &svcapitypes.LaunchTemplateHibernationOptionsRequest{}
				if elem.LaunchTemplateData.HibernationOptions.Configured != nil {
					f3f10.Configured = elem.LaunchTemplateData.HibernationOptions.Configured
				}
				f3.HibernationOptions = f3f10
			}
			if elem.LaunchTemplateData.IamInstanceProfile != nil {
				f3f11 := &svcapitypes.LaunchTemplateIAMInstanceProfileSpecificationRequest{}
				if elem.LaunchTemplateData.IamInstanceProfile.Arn != nil {
					f3f11.ARN = elem.LaunchTemplateData.IamInstanceProfile.Arn
				}
				if elem.LaunchTemplateData.IamInstanceProfile.Name != nil {
					f3f11.Name = elem.LaunchTemplateData.IamInstanceProfile.Name
				}
				f3.IAMInstanceProfile = f3f11
			}
			if elem.LaunchTemplateData.ImageId != nil {
				f3.ImageID = elem.LaunchTemplateData.ImageId
			}
			if elem.LaunchTemplateData.InstanceInitiatedShutdownBehavior != "" {
				f3.InstanceInitiatedShutdownBehavior = aws.String(string(elem.LaunchTemplateData.InstanceInitiatedShutdownBehavior))
			}
			if elem.LaunchTemplateData.InstanceMarketOptions != nil {
				f3f14 := &svcapitypes.LaunchTemplateInstanceMarketOptionsRequest{}
				if elem.LaunchTemplateData.InstanceMarketOptions.MarketType != "" {
					f3f14.MarketType = aws.String(string(elem.LaunchTemplateData.InstanceMarketOptions.MarketType))
				}
				if elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions != nil {
					f3f14f1 := &svcapitypes.LaunchTemplateSpotMarketOptionsRequest{}
					if elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.BlockDurationMinutes != nil {
						blockDurationMinutesCopy := int64(*elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.BlockDurationMinutes)
						f3f14f1.BlockDurationMinutes = &blockDurationMinutesCopy
					}
					if elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior != "" {
						f3f14f1.InstanceInterruptionBehavior = aws.String(string(elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior))
					}
					if elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice != nil {
						f3f14f1.MaxPrice = elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice
					}
					if elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SpotInstanceType != "" {
						f3f14f1.SpotInstanceType = aws.String(string(elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SpotInstanceType))
					}
					if elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.ValidUntil != nil {
						f3f14f1.ValidUntil = &metav1.Time{*elem.LaunchTemplateData.InstanceMarketOptions.SpotOptions.ValidUntil}
					}
					f3f14.SpotOptions = f3f14f1
				}
				f3.InstanceMarketOptions = f3f14
			}
			if elem.LaunchTemplateData.InstanceRequirements != nil {
				f3f15 := &svcapitypes.InstanceRequirementsRequest{}
				if elem.LaunchTemplateData.InstanceRequirements.AcceleratorCount != nil {
					f3f15f0 := &svcapitypes.AcceleratorCountRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Max != nil {
						maxCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Max)
						f3f15f0.Max = &maxCopy
					}
					if elem.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Min != nil {
						minCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Min)
						f3f15f0.Min = &minCopy
					}
					f3f15.AcceleratorCount = f3f15f0
				}
				if elem.LaunchTemplateData.InstanceRequirements.AcceleratorManufacturers != nil {
					f3f15f1 := []*string{}
					for _, f3f15f1iter := range elem.LaunchTemplateData.InstanceRequirements.AcceleratorManufacturers {
						var f3f15f1elem *string
						f3f15f1elem = aws.String(string(f3f15f1iter))
						f3f15f1 = append(f3f15f1, f3f15f1elem)
					}
					f3f15.AcceleratorManufacturers = f3f15f1
				}
				if elem.LaunchTemplateData.InstanceRequirements.AcceleratorNames != nil {
					f3f15f2 := []*string{}
					for _, f3f15f2iter := range elem.LaunchTemplateData.InstanceRequirements.AcceleratorNames {
						var f3f15f2elem *string
						f3f15f2elem = aws.String(string(f3f15f2iter))
						f3f15f2 = append(f3f15f2, f3f15f2elem)
					}
					f3f15.AcceleratorNames = f3f15f2
				}
				if elem.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB != nil {
					f3f15f3 := &svcapitypes.AcceleratorTotalMemoryMiBRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Max != nil {
						maxCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Max)
						f3f15f3.Max = &maxCopy
					}
					if elem.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Min != nil {
						minCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Min)
						f3f15f3.Min = &minCopy
					}
					f3f15.AcceleratorTotalMemoryMiB = f3f15f3
				}
				if elem.LaunchTemplateData.InstanceRequirements.AcceleratorTypes != nil {
					f3f15f4 := []*string{}
					for _, f3f15f4iter := range elem.LaunchTemplateData.InstanceRequirements.AcceleratorTypes {
						var f3f15f4elem *string
						f3f15f4elem = aws.String(string(f3f15f4iter))
						f3f15f4 = append(f3f15f4, f3f15f4elem)
					}
					f3f15.AcceleratorTypes = f3f15f4
				}
				if elem.LaunchTemplateData.InstanceRequirements.AllowedInstanceTypes != nil {
					f3f15.AllowedInstanceTypes = aws.StringSlice(elem.LaunchTemplateData.InstanceRequirements.AllowedInstanceTypes)
				}
				if elem.LaunchTemplateData.InstanceRequirements.BareMetal != "" {
					f3f15.BareMetal = aws.String(string(elem.LaunchTemplateData.InstanceRequirements.BareMetal))
				}
				if elem.LaunchTemplateData.InstanceRequirements.BaselineEbsBandwidthMbps != nil {
					f3f15f7 := &svcapitypes.BaselineEBSBandwidthMbpsRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.BaselineEbsBandwidthMbps.Max != nil {
						maxCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.BaselineEbsBandwidthMbps.Max)
						f3f15f7.Max = &maxCopy
					}
					if elem.LaunchTemplateData.InstanceRequirements.BaselineEbsBandwidthMbps.Min != nil {
						minCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.BaselineEbsBandwidthMbps.Min)
						f3f15f7.Min = &minCopy
					}
					f3f15.BaselineEBSBandwidthMbps = f3f15f7
				}
				if elem.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors != nil {
					f3f15f8 := &svcapitypes.BaselinePerformanceFactorsRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors.Cpu != nil {
						f3f15f8f0 := &svcapitypes.CPUPerformanceFactorRequest{}
						if elem.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors.Cpu.References != nil {
							f3f15f8f0f0 := []*svcapitypes.PerformanceFactorReferenceRequest{}
							for _, f3f15f8f0f0iter := range elem.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors.Cpu.References {
								f3f15f8f0f0elem := &svcapitypes.PerformanceFactorReferenceRequest{}
								if f3f15f8f0f0iter.InstanceFamily != nil {
									f3f15f8f0f0elem.InstanceFamily = f3f15f8f0f0iter.InstanceFamily
								}
								f3f15f8f0f0 = append(f3f15f8f0f0, f3f15f8f0f0elem)
							}
							f3f15f8f0.References = f3f15f8f0f0
						}
						f3f15f8.CPU = f3f15f8f0
					}
					f3f15.BaselinePerformanceFactors = f3f15f8
				}
				if elem.LaunchTemplateData.InstanceRequirements.BurstablePerformance != "" {
					f3f15.BurstablePerformance = aws.String(string(elem.LaunchTemplateData.InstanceRequirements.BurstablePerformance))
				}
				if elem.LaunchTemplateData.InstanceRequirements.CpuManufacturers != nil {
					f3f15f10 := []*string{}
					for _, f3f15f10iter := range elem.LaunchTemplateData.InstanceRequirements.CpuManufacturers {
						var f3f15f10elem *string
						f3f15f10elem = aws.String(string(f3f15f10iter))
						f3f15f10 = append(f3f15f10, f3f15f10elem)
					}
					f3f15.CPUManufacturers = f3f15f10
				}
				if elem.LaunchTemplateData.InstanceRequirements.ExcludedInstanceTypes != nil {
					f3f15.ExcludedInstanceTypes = aws.StringSlice(elem.LaunchTemplateData.InstanceRequirements.ExcludedInstanceTypes)
				}
				if elem.LaunchTemplateData.InstanceRequirements.InstanceGenerations != nil {
					f3f15f12 := []*string{}
					for _, f3f15f12iter := range elem.LaunchTemplateData.InstanceRequirements.InstanceGenerations {
						var f3f15f12elem *string
						f3f15f12elem = aws.String(string(f3f15f12iter))
						f3f15f12 = append(f3f15f12, f3f15f12elem)
					}
					f3f15.InstanceGenerations = f3f15f12
				}
				if elem.LaunchTemplateData.InstanceRequirements.LocalStorage != "" {
					f3f15.LocalStorage = aws.String(string(elem.LaunchTemplateData.InstanceRequirements.LocalStorage))
				}
				if elem.LaunchTemplateData.InstanceRequirements.LocalStorageTypes != nil {
					f3f15f14 := []*string{}
					for _, f3f15f14iter := range elem.LaunchTemplateData.InstanceRequirements.LocalStorageTypes {
						var f3f15f14elem *string
						f3f15f14elem = aws.String(string(f3f15f14iter))
						f3f15f14 = append(f3f15f14, f3f15f14elem)
					}
					f3f15.LocalStorageTypes = f3f15f14
				}
				if elem.LaunchTemplateData.InstanceRequirements.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice != nil {
					maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice)
					f3f15.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice = &maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy
				}
				if elem.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCpu != nil {
					f3f15f16 := &svcapitypes.MemoryGiBPerVCPURequest{}
					if elem.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCpu.Max != nil {
						f3f15f16.Max = elem.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCpu.Max
					}
					if elem.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCpu.Min != nil {
						f3f15f16.Min = elem.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCpu.Min
					}
					f3f15.MemoryGiBPerVCPU = f3f15f16
				}
				if elem.LaunchTemplateData.InstanceRequirements.MemoryMiB != nil {
					f3f15f17 := &svcapitypes.MemoryMiBRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.MemoryMiB.Max != nil {
						maxCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.MemoryMiB.Max)
						f3f15f17.Max = &maxCopy
					}
					if elem.LaunchTemplateData.InstanceRequirements.MemoryMiB.Min != nil {
						minCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.MemoryMiB.Min)
						f3f15f17.Min = &minCopy
					}
					f3f15.MemoryMiB = f3f15f17
				}
				if elem.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps != nil {
					f3f15f18 := &svcapitypes.NetworkBandwidthGbpsRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Max != nil {
						f3f15f18.Max = elem.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Max
					}
					if elem.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Min != nil {
						f3f15f18.Min = elem.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Min
					}
					f3f15.NetworkBandwidthGbps = f3f15f18
				}
				if elem.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount != nil {
					f3f15f19 := &svcapitypes.NetworkInterfaceCountRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Max != nil {
						maxCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Max)
						f3f15f19.Max = &maxCopy
					}
					if elem.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Min != nil {
						minCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Min)
						f3f15f19.Min = &minCopy
					}
					f3f15.NetworkInterfaceCount = f3f15f19
				}
				if elem.LaunchTemplateData.InstanceRequirements.OnDemandMaxPricePercentageOverLowestPrice != nil {
					onDemandMaxPricePercentageOverLowestPriceCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.OnDemandMaxPricePercentageOverLowestPrice)
					f3f15.OnDemandMaxPricePercentageOverLowestPrice = &onDemandMaxPricePercentageOverLowestPriceCopy
				}
				if elem.LaunchTemplateData.InstanceRequirements.RequireHibernateSupport != nil {
					f3f15.RequireHibernateSupport = elem.LaunchTemplateData.InstanceRequirements.RequireHibernateSupport
				}
				if elem.LaunchTemplateData.InstanceRequirements.SpotMaxPricePercentageOverLowestPrice != nil {
					spotMaxPricePercentageOverLowestPriceCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.SpotMaxPricePercentageOverLowestPrice)
					f3f15.SpotMaxPricePercentageOverLowestPrice = &spotMaxPricePercentageOverLowestPriceCopy
				}
				if elem.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB != nil {
					f3f15f23 := &svcapitypes.TotalLocalStorageGBRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Max != nil {
						f3f15f23.Max = elem.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Max
					}
					if elem.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Min != nil {
						f3f15f23.Min = elem.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Min
					}
					f3f15.TotalLocalStorageGB = f3f15f23
				}
				if elem.LaunchTemplateData.InstanceRequirements.VCpuCount != nil {
					f3f15f24 := &svcapitypes.VCPUCountRangeRequest{}
					if elem.LaunchTemplateData.InstanceRequirements.VCpuCount.Max != nil {
						maxCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.VCpuCount.Max)
						f3f15f24.Max = &maxCopy
					}
					if elem.LaunchTemplateData.InstanceRequirements.VCpuCount.Min != nil {
						minCopy := int64(*elem.LaunchTemplateData.InstanceRequirements.VCpuCount.Min)
						f3f15f24.Min = &minCopy
					}
					f3f15.VCPUCount = f3f15f24
				}
				f3.InstanceRequirements = f3f15
			}
			if elem.LaunchTemplateData.InstanceType != "" {
				f3.InstanceType = aws.String(string(elem.LaunchTemplateData.InstanceType))
			}
			if elem.LaunchTemplateData.KernelId != nil {
				f3.KernelID = elem.LaunchTemplateData.KernelId
			}
			if elem.LaunchTemplateData.KeyName != nil {
				f3.KeyName = elem.LaunchTemplateData.KeyName
			}
			if elem.LaunchTemplateData.LicenseSpecifications != nil {
				f3f19 := []*svcapitypes.LaunchTemplateLicenseConfigurationRequest{}
				for _, f3f19iter := range elem.LaunchTemplateData.LicenseSpecifications {
					f3f19elem := &svcapitypes.LaunchTemplateLicenseConfigurationRequest{}
					if f3f19iter.LicenseConfigurationArn != nil {
						f3f19elem.LicenseConfigurationARN = f3f19iter.LicenseConfigurationArn
					}
					f3f19 = append(f3f19, f3f19elem)
				}
				f3.LicenseSpecifications = f3f19
			}
			if elem.LaunchTemplateData.MaintenanceOptions != nil {
				f3f20 := &svcapitypes.LaunchTemplateInstanceMaintenanceOptionsRequest{}
				if elem.LaunchTemplateData.MaintenanceOptions.AutoRecovery != "" {
					f3f20.AutoRecovery = aws.String(string(elem.LaunchTemplateData.MaintenanceOptions.AutoRecovery))
				}
				f3.MaintenanceOptions = f3f20
			}
			if elem.LaunchTemplateData.MetadataOptions != nil {
				f3f21 := &svcapitypes.LaunchTemplateInstanceMetadataOptionsRequest{}
				if elem.LaunchTemplateData.MetadataOptions.HttpEndpoint != "" {
					f3f21.HTTPEndpoint = aws.String(string(elem.LaunchTemplateData.MetadataOptions.HttpEndpoint))
				}
				if elem.LaunchTemplateData.MetadataOptions.HttpProtocolIpv6 != "" {
					f3f21.HTTPProtocolIPv6 = aws.String(string(elem.LaunchTemplateData.MetadataOptions.HttpProtocolIpv6))
				}
				if elem.LaunchTemplateData.MetadataOptions.HttpPutResponseHopLimit != nil {
					httpPutResponseHopLimitCopy := int64(*elem.LaunchTemplateData.MetadataOptions.HttpPutResponseHopLimit)
					f3f21.HTTPPutResponseHopLimit = &httpPutResponseHopLimitCopy
				}
				if elem.LaunchTemplateData.MetadataOptions.HttpTokens != "" {
					f3f21.HTTPTokens = aws.String(string(elem.LaunchTemplateData.MetadataOptions.HttpTokens))
				}
				if elem.LaunchTemplateData.MetadataOptions.InstanceMetadataTags != "" {
					f3f21.InstanceMetadataTags = aws.String(string(elem.LaunchTemplateData.MetadataOptions.InstanceMetadataTags))
				}
				f3.MetadataOptions = f3f21
			}
			if elem.LaunchTemplateData.Monitoring != nil {
				f3f22 := &svcapitypes.LaunchTemplatesMonitoringRequest{}
				if elem.LaunchTemplateData.Monitoring.Enabled != nil {
					f3f22.Enabled = elem.LaunchTemplateData.Monitoring.Enabled
				}
				f3.Monitoring = f3f22
			}
			if elem.LaunchTemplateData.NetworkInterfaces != nil {
				f3f23 := []*svcapitypes.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{}
				for _, f3f23iter := range elem.LaunchTemplateData.NetworkInterfaces {
					f3f23elem := &svcapitypes.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{}
					if f3f23iter.AssociateCarrierIpAddress != nil {
						f3f23elem.AssociateCarrierIPAddress = f3f23iter.AssociateCarrierIpAddress
					}
					if f3f23iter.AssociatePublicIpAddress != nil {
						f3f23elem.AssociatePublicIPAddress = f3f23iter.AssociatePublicIpAddress
					}
					if f3f23iter.DeleteOnTermination != nil {
						f3f23elem.DeleteOnTermination = f3f23iter.DeleteOnTermination
					}
					if f3f23iter.Description != nil {
						f3f23elem.Description = f3f23iter.Description
					}
					if f3f23iter.DeviceIndex != nil {
						deviceIndexCopy := int64(*f3f23iter.DeviceIndex)
						f3f23elem.DeviceIndex = &deviceIndexCopy
					}
					if f3f23iter.Groups != nil {
						f3f23elem.Groups = aws.StringSlice(f3f23iter.Groups)
					}
					if f3f23iter.InterfaceType != nil {
						f3f23elem.InterfaceType = f3f23iter.InterfaceType
					}
					if f3f23iter.Ipv4PrefixCount != nil {
						ipv4PrefixCountCopy := int64(*f3f23iter.Ipv4PrefixCount)
						f3f23elem.IPv4PrefixCount = &ipv4PrefixCountCopy
					}
					if f3f23iter.Ipv4Prefixes != nil {
						f3f23elemf10 := []*svcapitypes.IPv4PrefixSpecificationRequest{}
						for _, f3f23elemf10iter := range f3f23iter.Ipv4Prefixes {
							f3f23elemf10elem := &svcapitypes.IPv4PrefixSpecificationRequest{}
							if f3f23elemf10iter.Ipv4Prefix != nil {
								f3f23elemf10elem.IPv4Prefix = f3f23elemf10iter.Ipv4Prefix
							}
							f3f23elemf10 = append(f3f23elemf10, f3f23elemf10elem)
						}
						f3f23elem.IPv4Prefixes = f3f23elemf10
					}
					if f3f23iter.Ipv6AddressCount != nil {
						ipv6AddressCountCopy := int64(*f3f23iter.Ipv6AddressCount)
						f3f23elem.IPv6AddressCount = &ipv6AddressCountCopy
					}
					if f3f23iter.Ipv6Addresses != nil {
						f3f23elemf12 := []*svcapitypes.InstanceIPv6AddressRequest{}
						for _, f3f23elemf12iter := range f3f23iter.Ipv6Addresses {
							f3f23elemf12elem := &svcapitypes.InstanceIPv6AddressRequest{}
							if f3f23elemf12iter.Ipv6Address != nil {
								f3f23elemf12elem.IPv6Address = f3f23elemf12iter.Ipv6Address
							}
							f3f23elemf12 = append(f3f23elemf12, f3f23elemf12elem)
						}
						f3f23elem.IPv6Addresses = f3f23elemf12
					}
					if f3f23iter.Ipv6PrefixCount != nil {
						ipv6PrefixCountCopy := int64(*f3f23iter.Ipv6PrefixCount)
						f3f23elem.IPv6PrefixCount = &ipv6PrefixCountCopy
					}
					if f3f23iter.Ipv6Prefixes != nil {
						f3f23elemf14 := []*svcapitypes.IPv6PrefixSpecificationRequest{}
						for _, f3f23elemf14iter := range f3f23iter.Ipv6Prefixes {
							f3f23elemf14elem := &svcapitypes.IPv6PrefixSpecificationRequest{}
							if f3f23elemf14iter.Ipv6Prefix != nil {
								f3f23elemf14elem.IPv6Prefix = f3f23elemf14iter.Ipv6Prefix
							}
							f3f23elemf14 = append(f3f23elemf14, f3f23elemf14elem)
						}
						f3f23elem.IPv6Prefixes = f3f23elemf14
					}
					if f3f23iter.NetworkCardIndex != nil {
						networkCardIndexCopy := int64(*f3f23iter.NetworkCardIndex)
						f3f23elem.NetworkCardIndex = &networkCardIndexCopy
					}
					if f3f23iter.NetworkInterfaceId != nil {
						f3f23elem.NetworkInterfaceID = f3f23iter.NetworkInterfaceId
					}
					if f3f23iter.PrimaryIpv6 != nil {
						f3f23elem.PrimaryIPv6 = f3f23iter.PrimaryIpv6
					}
					if f3f23iter.PrivateIpAddress != nil {
						f3f23elem.PrivateIPAddress = f3f23iter.PrivateIpAddress
					}
					if f3f23iter.PrivateIpAddresses != nil {
						f3f23elemf19 := []*svcapitypes.PrivateIPAddressSpecification{}
						for _, f3f23elemf19iter := range f3f23iter.PrivateIpAddresses {
							f3f23elemf19elem := &svcapitypes.PrivateIPAddressSpecification{}
							if f3f23elemf19iter.Primary != nil {
								f3f23elemf19elem.Primary = f3f23elemf19iter.Primary
							}
							if f3f23elemf19iter.PrivateIpAddress != nil {
								f3f23elemf19elem.PrivateIPAddress = f3f23elemf19iter.PrivateIpAddress
							}
							f3f23elemf19 = append(f3f23elemf19, f3f23elemf19elem)
						}
						f3f23elem.PrivateIPAddresses = f3f23elemf19
					}
					if f3f23iter.SecondaryPrivateIpAddressCount != nil {
						secondaryPrivateIPAddressCountCopy := int64(*f3f23iter.SecondaryPrivateIpAddressCount)
						f3f23elem.SecondaryPrivateIPAddressCount = &secondaryPrivateIPAddressCountCopy
					}
					if f3f23iter.SubnetId != nil {
						f3f23elem.SubnetID = f3f23iter.SubnetId
					}
					f3f23 = append(f3f23, f3f23elem)
				}
				f3.NetworkInterfaces = f3f23
			}
			if elem.LaunchTemplateData.Placement != nil {
				f3f25 := &svcapitypes.LaunchTemplatePlacementRequest{}
				if elem.LaunchTemplateData.Placement.Affinity != nil {
					f3f25.Affinity = elem.LaunchTemplateData.Placement.Affinity
				}
				if elem.LaunchTemplateData.Placement.AvailabilityZone != nil {
					f3f25.AvailabilityZone = elem.LaunchTemplateData.Placement.AvailabilityZone
				}
				if elem.LaunchTemplateData.Placement.GroupId != nil {
					f3f25.GroupID = elem.LaunchTemplateData.Placement.GroupId
				}
				if elem.LaunchTemplateData.Placement.GroupName != nil {
					f3f25.GroupName = elem.LaunchTemplateData.Placement.GroupName
				}
				if elem.LaunchTemplateData.Placement.HostId != nil {
					f3f25.HostID = elem.LaunchTemplateData.Placement.HostId
				}
				if elem.LaunchTemplateData.Placement.HostResourceGroupArn != nil {
					f3f25.HostResourceGroupARN = elem.LaunchTemplateData.Placement.HostResourceGroupArn
				}
				if elem.LaunchTemplateData.Placement.PartitionNumber != nil {
					partitionNumberCopy := int64(*elem.LaunchTemplateData.Placement.PartitionNumber)
					f3f25.PartitionNumber = &partitionNumberCopy
				}
				if elem.LaunchTemplateData.Placement.SpreadDomain != nil {
					f3f25.SpreadDomain = elem.LaunchTemplateData.Placement.SpreadDomain
				}
				if elem.LaunchTemplateData.Placement.Tenancy != "" {
					f3f25.Tenancy = aws.String(string(elem.LaunchTemplateData.Placement.Tenancy))
				}
				f3.Placement = f3f25
			}
			if elem.LaunchTemplateData.PrivateDnsNameOptions != nil {
				f3f26 := &svcapitypes.LaunchTemplatePrivateDNSNameOptionsRequest{}
				if elem.LaunchTemplateData.PrivateDnsNameOptions.EnableResourceNameDnsAAAARecord != nil {
					f3f26.EnableResourceNameDNSAAAARecord = elem.LaunchTemplateData.PrivateDnsNameOptions.EnableResourceNameDnsAAAARecord
				}
				if elem.LaunchTemplateData.PrivateDnsNameOptions.EnableResourceNameDnsARecord != nil {
					f3f26.EnableResourceNameDNSARecord = elem.LaunchTemplateData.PrivateDnsNameOptions.EnableResourceNameDnsARecord
				}
				if elem.LaunchTemplateData.PrivateDnsNameOptions.HostnameType != "" {
					f3f26.HostnameType = aws.String(string(elem.LaunchTemplateData.PrivateDnsNameOptions.HostnameType))
				}
				f3.PrivateDNSNameOptions = f3f26
			}
			if elem.LaunchTemplateData.RamDiskId != nil {
				f3.RAMDiskID = elem.LaunchTemplateData.RamDiskId
			}
			if elem.LaunchTemplateData.SecurityGroupIds != nil {
				f3.SecurityGroupIDs = aws.StringSlice(elem.LaunchTemplateData.SecurityGroupIds)
			}
			if elem.LaunchTemplateData.SecurityGroups != nil {
				f3.SecurityGroups = aws.StringSlice(elem.LaunchTemplateData.SecurityGroups)
			}
			if elem.LaunchTemplateData.UserData != nil {
				f3.UserData = elem.LaunchTemplateData.UserData
			}
			ko.Spec.Data = f3
		} else {
			ko.Spec.Data = nil
		}
		if elem.Operator != nil {
			f6 := &svcapitypes.OperatorResponse{}
			if elem.Operator.Managed != nil {
				f6.Managed = elem.Operator.Managed
			}
			if elem.Operator.Principal != nil {
				f6.Principal = elem.Operator.Principal
			}
			ko.Status.Operator = f6
		} else {
			ko.Status.Operator = nil
		}
		if elem.VersionDescription != nil {
			ko.Spec.VersionDescription = elem.VersionDescription
		} else {
			ko.Spec.VersionDescription = nil
		}
		break
	}
	return nil
}

func newListLaunchTemplateVersionRequestPayload(
	r *resource,
) (*svcsdk.DescribeLaunchTemplateVersionsInput, error) {
	res := &svcsdk.DescribeLaunchTemplateVersionsInput{}
	if r.ko.Status.ID != nil {
		res.LaunchTemplateId = r.ko.Status.ID
	}
	if r.ko.Spec.DefaultVersion != nil {
		res.Versions = []string{fmt.Sprintf("%d", *r.ko.Spec.DefaultVersion)}
	}

	return res, nil
}

// updateDefaultVersion patches the supplied resource in the backend AWS service API
func (rm *resourceManager) updateDefaultVersion(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateDefaultVersion")
	defer func() {
		exit(err)
	}()

	// TODO (michaelhtm) Not sure if we should check if
	// the defaultVersion is greater than latestVersion
	//
	// My proposal would be to return a terminal error
	// since the launchTemplate's latestVersion will never
	// increase intil we make updates...˘\\/(ヅ)\/˘˘
	// if *desired.ko.Spec.DefaultVersion > *latest.ko.Status.LatestVersion {
	// 	return ackerr.NewTerminalError(fmt.Errorf("desired version number is ahead of the latest version"))
	// }

	ko := desired.ko.DeepCopy()

	input, err := newUpdateLaunchTemplateVersionRequestPayload(&resource{ko})
	if err != nil {
		return err
	}

	var resp *svcsdk.ModifyLaunchTemplateOutput
	resp, err = rm.sdkapi.ModifyLaunchTemplate(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyLaunchTemplate", err)
	if err != nil {
		return err
	}

	if resp.LaunchTemplate.CreateTime != nil {
		ko.Status.CreateTime = &metav1.Time{*resp.LaunchTemplate.CreateTime}
	} else {
		ko.Status.CreateTime = nil
	}
	if resp.LaunchTemplate.CreatedBy != nil {
		ko.Status.CreatedBy = resp.LaunchTemplate.CreatedBy
	} else {
		ko.Status.CreatedBy = nil
	}
	if resp.LaunchTemplate.DefaultVersionNumber != nil {
		ko.Spec.DefaultVersion = resp.LaunchTemplate.DefaultVersionNumber
	} else {
		ko.Spec.DefaultVersion = nil
	}
	if resp.LaunchTemplate.LatestVersionNumber != nil {
		ko.Status.LatestVersion = resp.LaunchTemplate.LatestVersionNumber
	} else {
		ko.Status.LatestVersion = nil
	}
	if resp.LaunchTemplate.Operator != nil {
		f6 := &svcapitypes.OperatorResponse{}
		if resp.LaunchTemplate.Operator.Managed != nil {
			f6.Managed = resp.LaunchTemplate.Operator.Managed
		}
		if resp.LaunchTemplate.Operator.Principal != nil {
			f6.Principal = resp.LaunchTemplate.Operator.Principal
		}
		ko.Status.Operator = f6
	} else {
		ko.Status.Operator = nil
	}
	if resp.LaunchTemplate.Tags != nil {
		f7 := []*svcapitypes.Tag{}
		for _, f7iter := range resp.LaunchTemplate.Tags {
			f7elem := &svcapitypes.Tag{}
			if f7iter.Key != nil {
				f7elem.Key = f7iter.Key
			}
			if f7iter.Value != nil {
				f7elem.Value = f7iter.Value
			}
			f7 = append(f7, f7elem)
		}
		ko.Spec.Tags = f7
	} else {
		ko.Spec.Tags = nil
	}

	rm.setStatusDefaults(ko)
	return nil
}

func newUpdateLaunchTemplateVersionRequestPayload(
	r *resource,
) (*svcsdk.ModifyLaunchTemplateInput, error) {
	res := &svcsdk.ModifyLaunchTemplateInput{}

	if r.ko.Spec.Name != nil {
		res.LaunchTemplateName = r.ko.Spec.Name
	}
	if r.ko.Spec.DefaultVersion != nil {
		res.DefaultVersion = aws.String(fmt.Sprintf("%d", *r.ko.Spec.DefaultVersion))
	}

	return res, nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateLaunchTemplate.TagSpecification
// and ensures the ResourceType is always set to 'launch-template'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateLaunchTemplateInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}

	if r.ko.Spec.Tags != nil {

		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {

			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				{
					tag.Key = desiredTag.Key
					tag.Value = desiredTag.Value
				}
				requestedTags = append(requestedTags, tag)
			}

		}
		desiredTagSpecs.ResourceType = svcsdktypes.ResourceTypeLaunchTemplate
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

func (rm *resourceManager) checkForMissingRequiredFields(r *resource) bool {
	return r.ko.Status.ID == nil
}

var syncTags = tags.Sync
