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
	"math"
	"reflect"
	"strconv"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func (rm *resourceManager) setDefaultTemplateVersion(r *resource, input *svcsdk.ModifyLaunchTemplateInput) error {

	newDefaultVersion := *r.ko.Status.LatestVersionNumber + 1
	input.DefaultVersion = aws.String(strconv.FormatInt(newDefaultVersion, 10))

	return nil
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

// syncTags used to keep tags in sync by calling Create and Delete API's
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func(err error) {
		exit(err)
	}(err)

	resourceId := []string{*latest.ko.Status.LaunchTemplateID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from launchtemplate resource", "tags", toDelete)
		_, err = rm.sdkapi.DeleteTags(
			ctx,
			&svcsdk.DeleteTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toDelete),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "DeleteTags", err)
		if err != nil {
			return err
		}

	}

	if len(toAdd) > 0 {
		rlog.Debug("adding tags to launchtemplate resource", "tags", toAdd)
		_, err = rm.sdkapi.CreateTags(
			ctx,
			&svcsdk.CreateTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "CreateTags", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// sdkTags converts *svcapitypes.Tag array to a *svcsdk.Tag array
func (rm *resourceManager) sdkTags(
	tags []svcapitypes.Tag,
) (sdktags []svcsdktypes.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(i)
		sdktags = append(sdktags, *sdktag)
	}

	return sdktags
}

func (rm *resourceManager) newTag(
	c svcapitypes.Tag,
) *svcsdktypes.Tag {
	res := &svcsdktypes.Tag{}
	if c.Key != nil {
		res.Key = c.Key
	}
	if c.Value != nil {
		res.Value = c.Value

	}

	return res
}

// computeTagsDelta returns tags to be added and removed from the resource
func computeTagsDelta(
	desired []*svcapitypes.Tag,
	latest []*svcapitypes.Tag,
) (toAdd []svcapitypes.Tag, toDelete []svcapitypes.Tag) {

	desiredTags := map[string]string{}
	for _, tag := range desired {
		desiredTags[*tag.Key] = *tag.Value
	}

	latestTags := map[string]string{}
	for _, tag := range latest {
		latestTags[*tag.Key] = *tag.Value
	}

	for _, tag := range desired {
		val, ok := latestTags[*tag.Key]
		if !ok || val != *tag.Value {
			toAdd = append(toAdd, *tag)
		}
	}

	for _, tag := range latest {
		_, ok := desiredTags[*tag.Key]
		if !ok {
			toDelete = append(toDelete, *tag)
		}
	}

	return toAdd, toDelete

}

func customPreCompare(delta *ackcompare.Delta, a *resource, b *resource) {

	if !reflect.DeepEqual(a.ko.Spec.VersionDescription, b.ko.Spec.VersionDescription) {
		delta.Add("Spec.VersionDescription", a.ko.Spec.VersionDescription, b.ko.Spec.VersionDescription)
	}
	if !reflect.DeepEqual(a.ko.Spec.LaunchTemplateData, b.ko.Spec.LaunchTemplateData) {
		delta.Add("Spec.LaunchTemplateData", a.ko.Spec.LaunchTemplateData, b.ko.Spec.LaunchTemplateData)
	}

}

// method to create new template version on every update of the launchtemplatedata
func (rm *resourceManager) CreateLaunchTemplateVersion(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.CreateLaunchTemplateVersion")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.CreateLaunchTemplateVersionInput{}
	input.LaunchTemplateId = r.ko.Status.LaunchTemplateID
	input.VersionDescription = r.ko.Spec.VersionDescription

	if r.ko.Spec.LaunchTemplateData != nil {
		f0 := &svcsdktypes.RequestLaunchTemplateData{}
		if r.ko.Spec.LaunchTemplateData.BlockDeviceMappings != nil {
			f0f0 := []svcsdktypes.LaunchTemplateBlockDeviceMappingRequest{}
			for _, f0f0iter := range r.ko.Spec.LaunchTemplateData.BlockDeviceMappings {
				f0f0elem := &svcsdktypes.LaunchTemplateBlockDeviceMappingRequest{}
				if f0f0iter.DeviceName != nil {
					f0f0elem.DeviceName = f0f0iter.DeviceName
				}
				if f0f0iter.EBS != nil {
					f0f0elemf1 := &svcsdktypes.LaunchTemplateEbsBlockDeviceRequest{}
					if f0f0iter.EBS.DeleteOnTermination != nil {
						f0f0elemf1.DeleteOnTermination = f0f0iter.EBS.DeleteOnTermination
					}
					if f0f0iter.EBS.Encrypted != nil {
						f0f0elemf1.Encrypted = f0f0iter.EBS.Encrypted
					}
					if f0f0iter.EBS.IOPS != nil {
						iopsCopy0 := *f0f0iter.EBS.IOPS
						if iopsCopy0 > math.MaxInt32 || iopsCopy0 < math.MinInt32 {
							return fmt.Errorf("error: field Iops is of type int32")
						}
						iopsCopy := int32(iopsCopy0)
						f0f0elemf1.Iops = &iopsCopy
					}
					if f0f0iter.EBS.KMSKeyID != nil {
						f0f0elemf1.KmsKeyId = f0f0iter.EBS.KMSKeyID
					}
					if f0f0iter.EBS.SnapshotID != nil {
						f0f0elemf1.SnapshotId = f0f0iter.EBS.SnapshotID
					}
					if f0f0iter.EBS.Throughput != nil {
						throughputCopy0 := *f0f0iter.EBS.Throughput
						if throughputCopy0 > math.MaxInt32 || throughputCopy0 < math.MinInt32 {
							return fmt.Errorf("error: field Throughput is of type int32")
						}
						throughputCopy := int32(throughputCopy0)
						f0f0elemf1.Throughput = &throughputCopy
					}
					if f0f0iter.EBS.VolumeSize != nil {
						volumeSizeCopy0 := *f0f0iter.EBS.VolumeSize
						if volumeSizeCopy0 > math.MaxInt32 || volumeSizeCopy0 < math.MinInt32 {
							return fmt.Errorf("error: field VolumeSize is of type int32")
						}
						volumeSizeCopy := int32(volumeSizeCopy0)
						f0f0elemf1.VolumeSize = &volumeSizeCopy
					}
					if f0f0iter.EBS.VolumeType != nil {
						f0f0elemf1.VolumeType = svcsdktypes.VolumeType(*f0f0iter.EBS.VolumeType)
					}
					f0f0elem.Ebs = f0f0elemf1
				}
				if f0f0iter.NoDevice != nil {
					f0f0elem.NoDevice = f0f0iter.NoDevice
				}
				if f0f0iter.VirtualName != nil {
					f0f0elem.VirtualName = f0f0iter.VirtualName
				}
				f0f0 = append(f0f0, *f0f0elem)
			}
			f0.BlockDeviceMappings = f0f0
		}
		if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification != nil {
			f0f1 := &svcsdktypes.LaunchTemplateCapacityReservationSpecificationRequest{}
			if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationPreference != nil {
				f0f1.CapacityReservationPreference = svcsdktypes.CapacityReservationPreference(*r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationPreference)
			}
			if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget != nil {
				f0f1f1 := &svcsdktypes.CapacityReservationTarget{}
				if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID != nil {
					f0f1f1.CapacityReservationId = r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID
				}
				if r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN != nil {
					f0f1f1.CapacityReservationResourceGroupArn = r.ko.Spec.LaunchTemplateData.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN
				}
				f0f1.CapacityReservationTarget = f0f1f1
			}
			f0.CapacityReservationSpecification = f0f1
		}
		if r.ko.Spec.LaunchTemplateData.CPUOptions != nil {
			f0f2 := &svcsdktypes.LaunchTemplateCpuOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.CPUOptions.AmdSevSnp != nil {
				f0f2.AmdSevSnp = svcsdktypes.AmdSevSnpSpecification(*r.ko.Spec.LaunchTemplateData.CPUOptions.AmdSevSnp)
			}
			if r.ko.Spec.LaunchTemplateData.CPUOptions.CoreCount != nil {
				coreCountCopy0 := *r.ko.Spec.LaunchTemplateData.CPUOptions.CoreCount
				if coreCountCopy0 > math.MaxInt32 || coreCountCopy0 < math.MinInt32 {
					return fmt.Errorf("error: field CoreCount is of type int32")
				}
				coreCountCopy := int32(coreCountCopy0)
				f0f2.CoreCount = &coreCountCopy
			}
			if r.ko.Spec.LaunchTemplateData.CPUOptions.ThreadsPerCore != nil {
				threadsPerCoreCopy0 := *r.ko.Spec.LaunchTemplateData.CPUOptions.ThreadsPerCore
				if threadsPerCoreCopy0 > math.MaxInt32 || threadsPerCoreCopy0 < math.MinInt32 {
					return fmt.Errorf("error: field ThreadsPerCore is of type int32")
				}
				threadsPerCoreCopy := int32(threadsPerCoreCopy0)
				f0f2.ThreadsPerCore = &threadsPerCoreCopy
			}
			f0.CpuOptions = f0f2
		}
		if r.ko.Spec.LaunchTemplateData.CreditSpecification != nil {
			f0f3 := &svcsdktypes.CreditSpecificationRequest{}
			if r.ko.Spec.LaunchTemplateData.CreditSpecification.CPUCredits != nil {
				f0f3.CpuCredits = r.ko.Spec.LaunchTemplateData.CreditSpecification.CPUCredits
			}
			f0.CreditSpecification = f0f3
		}
		if r.ko.Spec.LaunchTemplateData.DisableAPIStop != nil {
			f0.DisableApiStop = r.ko.Spec.LaunchTemplateData.DisableAPIStop
		}
		if r.ko.Spec.LaunchTemplateData.DisableAPITermination != nil {
			f0.DisableApiTermination = r.ko.Spec.LaunchTemplateData.DisableAPITermination
		}
		if r.ko.Spec.LaunchTemplateData.EBSOptimized != nil {
			f0.EbsOptimized = r.ko.Spec.LaunchTemplateData.EBSOptimized
		}
		if r.ko.Spec.LaunchTemplateData.ElasticGPUSpecifications != nil {
			f0f7 := []svcsdktypes.ElasticGpuSpecification{}
			for _, f0f7iter := range r.ko.Spec.LaunchTemplateData.ElasticGPUSpecifications {
				f0f7elem := &svcsdktypes.ElasticGpuSpecification{}
				if f0f7iter.Type != nil {
					f0f7elem.Type = f0f7iter.Type
				}
				f0f7 = append(f0f7, *f0f7elem)
			}
			f0.ElasticGpuSpecifications = f0f7
		}
		if r.ko.Spec.LaunchTemplateData.ElasticInferenceAccelerators != nil {
			f0f8 := []svcsdktypes.LaunchTemplateElasticInferenceAccelerator{}
			for _, f0f8iter := range r.ko.Spec.LaunchTemplateData.ElasticInferenceAccelerators {
				f0f8elem := &svcsdktypes.LaunchTemplateElasticInferenceAccelerator{}
				if f0f8iter.Count != nil {
					countCopy0 := *f0f8iter.Count
					if countCopy0 > math.MaxInt32 || countCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Count is of type int32")
					}
					countCopy := int32(countCopy0)
					f0f8elem.Count = &countCopy
				}
				if f0f8iter.Type != nil {
					f0f8elem.Type = f0f8iter.Type
				}
				f0f8 = append(f0f8, *f0f8elem)
			}
			f0.ElasticInferenceAccelerators = f0f8
		}
		if r.ko.Spec.LaunchTemplateData.EnclaveOptions != nil {
			f0f9 := &svcsdktypes.LaunchTemplateEnclaveOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.EnclaveOptions.Enabled != nil {
				f0f9.Enabled = r.ko.Spec.LaunchTemplateData.EnclaveOptions.Enabled
			}
			f0.EnclaveOptions = f0f9
		}
		if r.ko.Spec.LaunchTemplateData.HibernationOptions != nil {
			f0f10 := &svcsdktypes.LaunchTemplateHibernationOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.HibernationOptions.Configured != nil {
				f0f10.Configured = r.ko.Spec.LaunchTemplateData.HibernationOptions.Configured
			}
			f0.HibernationOptions = f0f10
		}
		if r.ko.Spec.LaunchTemplateData.IAMInstanceProfile != nil {
			f0f11 := &svcsdktypes.LaunchTemplateIamInstanceProfileSpecificationRequest{}
			if r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.ARN != nil {
				f0f11.Arn = r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.ARN
			}
			if r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.Name != nil {
				f0f11.Name = r.ko.Spec.LaunchTemplateData.IAMInstanceProfile.Name
			}
			f0.IamInstanceProfile = f0f11
		}
		if r.ko.Spec.LaunchTemplateData.ImageID != nil {
			f0.ImageId = r.ko.Spec.LaunchTemplateData.ImageID
		}
		if r.ko.Spec.LaunchTemplateData.InstanceInitiatedShutdownBehavior != nil {
			f0.InstanceInitiatedShutdownBehavior = svcsdktypes.ShutdownBehavior(*r.ko.Spec.LaunchTemplateData.InstanceInitiatedShutdownBehavior)
		}
		if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions != nil {
			f0f14 := &svcsdktypes.LaunchTemplateInstanceMarketOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.MarketType != nil {
				f0f14.MarketType = svcsdktypes.MarketType(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.MarketType)
			}
			if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions != nil {
				f0f14f1 := &svcsdktypes.LaunchTemplateSpotMarketOptionsRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.BlockDurationMinutes != nil {
					blockDurationMinutesCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.BlockDurationMinutes
					if blockDurationMinutesCopy0 > math.MaxInt32 || blockDurationMinutesCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field BlockDurationMinutes is of type int32")
					}
					blockDurationMinutesCopy := int32(blockDurationMinutesCopy0)
					f0f14f1.BlockDurationMinutes = &blockDurationMinutesCopy
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior != nil {
					f0f14f1.InstanceInterruptionBehavior = svcsdktypes.InstanceInterruptionBehavior(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior)
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice != nil {
					f0f14f1.MaxPrice = r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.MaxPrice
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SpotInstanceType != nil {
					f0f14f1.SpotInstanceType = svcsdktypes.SpotInstanceType(*r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.SpotInstanceType)
				}
				if r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.ValidUntil != nil {
					f0f14f1.ValidUntil = &r.ko.Spec.LaunchTemplateData.InstanceMarketOptions.SpotOptions.ValidUntil.Time
				}
				f0f14.SpotOptions = f0f14f1
			}
			f0.InstanceMarketOptions = f0f14
		}
		if r.ko.Spec.LaunchTemplateData.InstanceRequirements != nil {
			f0f15 := &svcsdktypes.InstanceRequirementsRequest{}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorCount != nil {
				f0f15f0 := &svcsdktypes.AcceleratorCountRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Max != nil {
					maxCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Max
					if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Max is of type int32")
					}
					maxCopy := int32(maxCopy0)
					f0f15f0.Max = &maxCopy
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Min != nil {
					minCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorCount.Min
					if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Min is of type int32")
					}
					minCopy := int32(minCopy0)
					f0f15f0.Min = &minCopy
				}
				f0f15.AcceleratorCount = f0f15f0
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorManufacturers != nil {
				f0f15f1 := []svcsdktypes.AcceleratorManufacturer{}
				for _, f0f15f1iter := range r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorManufacturers {
					var f0f15f1elem string
					f0f15f1elem = string(*f0f15f1iter)
					f0f15f1 = append(f0f15f1, svcsdktypes.AcceleratorManufacturer(f0f15f1elem))
				}
				f0f15.AcceleratorManufacturers = f0f15f1
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorNames != nil {
				f0f15f2 := []svcsdktypes.AcceleratorName{}
				for _, f0f15f2iter := range r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorNames {
					var f0f15f2elem string
					f0f15f2elem = string(*f0f15f2iter)
					f0f15f2 = append(f0f15f2, svcsdktypes.AcceleratorName(f0f15f2elem))
				}
				f0f15.AcceleratorNames = f0f15f2
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB != nil {
				f0f15f3 := &svcsdktypes.AcceleratorTotalMemoryMiBRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Max != nil {
					maxCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Max
					if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Max is of type int32")
					}
					maxCopy := int32(maxCopy0)
					f0f15f3.Max = &maxCopy
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Min != nil {
					minCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorTotalMemoryMiB.Min
					if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Min is of type int32")
					}
					minCopy := int32(minCopy0)
					f0f15f3.Min = &minCopy
				}
				f0f15.AcceleratorTotalMemoryMiB = f0f15f3
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorTypes != nil {
				f0f15f4 := []svcsdktypes.AcceleratorType{}
				for _, f0f15f4iter := range r.ko.Spec.LaunchTemplateData.InstanceRequirements.AcceleratorTypes {
					var f0f15f4elem string
					f0f15f4elem = string(*f0f15f4iter)
					f0f15f4 = append(f0f15f4, svcsdktypes.AcceleratorType(f0f15f4elem))
				}
				f0f15.AcceleratorTypes = f0f15f4
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.AllowedInstanceTypes != nil {
				f0f15.AllowedInstanceTypes = aws.ToStringSlice(r.ko.Spec.LaunchTemplateData.InstanceRequirements.AllowedInstanceTypes)
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BareMetal != nil {
				f0f15.BareMetal = svcsdktypes.BareMetal(*r.ko.Spec.LaunchTemplateData.InstanceRequirements.BareMetal)
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselineEBSBandwidthMbps != nil {
				f0f15f7 := &svcsdktypes.BaselineEbsBandwidthMbpsRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselineEBSBandwidthMbps.Max != nil {
					maxCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselineEBSBandwidthMbps.Max
					if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Max is of type int32")
					}
					maxCopy := int32(maxCopy0)
					f0f15f7.Max = &maxCopy
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselineEBSBandwidthMbps.Min != nil {
					minCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselineEBSBandwidthMbps.Min
					if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Min is of type int32")
					}
					minCopy := int32(minCopy0)
					f0f15f7.Min = &minCopy
				}
				f0f15.BaselineEbsBandwidthMbps = f0f15f7
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors != nil {
				f0f15f8 := &svcsdktypes.BaselinePerformanceFactorsRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors.CPU != nil {
					f0f15f8f0 := &svcsdktypes.CpuPerformanceFactorRequest{}
					if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors.CPU.References != nil {
						f0f15f8f0f0 := []svcsdktypes.PerformanceFactorReferenceRequest{}
						for _, f0f15f8f0f0iter := range r.ko.Spec.LaunchTemplateData.InstanceRequirements.BaselinePerformanceFactors.CPU.References {
							f0f15f8f0f0elem := &svcsdktypes.PerformanceFactorReferenceRequest{}
							if f0f15f8f0f0iter.InstanceFamily != nil {
								f0f15f8f0f0elem.InstanceFamily = f0f15f8f0f0iter.InstanceFamily
							}
							f0f15f8f0f0 = append(f0f15f8f0f0, *f0f15f8f0f0elem)
						}
						f0f15f8f0.References = f0f15f8f0f0
					}
					f0f15f8.Cpu = f0f15f8f0
				}
				f0f15.BaselinePerformanceFactors = f0f15f8
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.BurstablePerformance != nil {
				f0f15.BurstablePerformance = svcsdktypes.BurstablePerformance(*r.ko.Spec.LaunchTemplateData.InstanceRequirements.BurstablePerformance)
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.CPUManufacturers != nil {
				f0f15f10 := []svcsdktypes.CpuManufacturer{}
				for _, f0f15f10iter := range r.ko.Spec.LaunchTemplateData.InstanceRequirements.CPUManufacturers {
					var f0f15f10elem string
					f0f15f10elem = string(*f0f15f10iter)
					f0f15f10 = append(f0f15f10, svcsdktypes.CpuManufacturer(f0f15f10elem))
				}
				f0f15.CpuManufacturers = f0f15f10
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.ExcludedInstanceTypes != nil {
				f0f15.ExcludedInstanceTypes = aws.ToStringSlice(r.ko.Spec.LaunchTemplateData.InstanceRequirements.ExcludedInstanceTypes)
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.InstanceGenerations != nil {
				f0f15f12 := []svcsdktypes.InstanceGeneration{}
				for _, f0f15f12iter := range r.ko.Spec.LaunchTemplateData.InstanceRequirements.InstanceGenerations {
					var f0f15f12elem string
					f0f15f12elem = string(*f0f15f12iter)
					f0f15f12 = append(f0f15f12, svcsdktypes.InstanceGeneration(f0f15f12elem))
				}
				f0f15.InstanceGenerations = f0f15f12
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.LocalStorage != nil {
				f0f15.LocalStorage = svcsdktypes.LocalStorage(*r.ko.Spec.LaunchTemplateData.InstanceRequirements.LocalStorage)
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.LocalStorageTypes != nil {
				f0f15f14 := []svcsdktypes.LocalStorageType{}
				for _, f0f15f14iter := range r.ko.Spec.LaunchTemplateData.InstanceRequirements.LocalStorageTypes {
					var f0f15f14elem string
					f0f15f14elem = string(*f0f15f14iter)
					f0f15f14 = append(f0f15f14, svcsdktypes.LocalStorageType(f0f15f14elem))
				}
				f0f15.LocalStorageTypes = f0f15f14
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice != nil {
				maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice
				if maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0 > math.MaxInt32 || maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0 < math.MinInt32 {
					return fmt.Errorf("error: field MaxSpotPriceAsPercentageOfOptimalOnDemandPrice is of type int32")
				}
				maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy := int32(maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy0)
				f0f15.MaxSpotPriceAsPercentageOfOptimalOnDemandPrice = &maxSpotPriceAsPercentageOfOptimalOnDemandPriceCopy
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCPU != nil {
				f0f15f16 := &svcsdktypes.MemoryGiBPerVCpuRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCPU.Max != nil {
					f0f15f16.Max = r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCPU.Max
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCPU.Min != nil {
					f0f15f16.Min = r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryGiBPerVCPU.Min
				}
				f0f15.MemoryGiBPerVCpu = f0f15f16
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryMiB != nil {
				f0f15f17 := &svcsdktypes.MemoryMiBRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryMiB.Max != nil {
					maxCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryMiB.Max
					if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Max is of type int32")
					}
					maxCopy := int32(maxCopy0)
					f0f15f17.Max = &maxCopy
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryMiB.Min != nil {
					minCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.MemoryMiB.Min
					if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Min is of type int32")
					}
					minCopy := int32(minCopy0)
					f0f15f17.Min = &minCopy
				}
				f0f15.MemoryMiB = f0f15f17
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps != nil {
				f0f15f18 := &svcsdktypes.NetworkBandwidthGbpsRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Max != nil {
					f0f15f18.Max = r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Max
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Min != nil {
					f0f15f18.Min = r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkBandwidthGbps.Min
				}
				f0f15.NetworkBandwidthGbps = f0f15f18
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount != nil {
				f0f15f19 := &svcsdktypes.NetworkInterfaceCountRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Max != nil {
					maxCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Max
					if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Max is of type int32")
					}
					maxCopy := int32(maxCopy0)
					f0f15f19.Max = &maxCopy
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Min != nil {
					minCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.NetworkInterfaceCount.Min
					if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Min is of type int32")
					}
					minCopy := int32(minCopy0)
					f0f15f19.Min = &minCopy
				}
				f0f15.NetworkInterfaceCount = f0f15f19
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.OnDemandMaxPricePercentageOverLowestPrice != nil {
				onDemandMaxPricePercentageOverLowestPriceCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.OnDemandMaxPricePercentageOverLowestPrice
				if onDemandMaxPricePercentageOverLowestPriceCopy0 > math.MaxInt32 || onDemandMaxPricePercentageOverLowestPriceCopy0 < math.MinInt32 {
					return fmt.Errorf("error: field OnDemandMaxPricePercentageOverLowestPrice is of type int32")
				}
				onDemandMaxPricePercentageOverLowestPriceCopy := int32(onDemandMaxPricePercentageOverLowestPriceCopy0)
				f0f15.OnDemandMaxPricePercentageOverLowestPrice = &onDemandMaxPricePercentageOverLowestPriceCopy
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.RequireHibernateSupport != nil {
				f0f15.RequireHibernateSupport = r.ko.Spec.LaunchTemplateData.InstanceRequirements.RequireHibernateSupport
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.SpotMaxPricePercentageOverLowestPrice != nil {
				spotMaxPricePercentageOverLowestPriceCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.SpotMaxPricePercentageOverLowestPrice
				if spotMaxPricePercentageOverLowestPriceCopy0 > math.MaxInt32 || spotMaxPricePercentageOverLowestPriceCopy0 < math.MinInt32 {
					return fmt.Errorf("error: field SpotMaxPricePercentageOverLowestPrice is of type int32")
				}
				spotMaxPricePercentageOverLowestPriceCopy := int32(spotMaxPricePercentageOverLowestPriceCopy0)
				f0f15.SpotMaxPricePercentageOverLowestPrice = &spotMaxPricePercentageOverLowestPriceCopy
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB != nil {
				f0f15f23 := &svcsdktypes.TotalLocalStorageGBRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Max != nil {
					f0f15f23.Max = r.ko.Spec.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Max
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Min != nil {
					f0f15f23.Min = r.ko.Spec.LaunchTemplateData.InstanceRequirements.TotalLocalStorageGB.Min
				}
				f0f15.TotalLocalStorageGB = f0f15f23
			}
			if r.ko.Spec.LaunchTemplateData.InstanceRequirements.VCPUCount != nil {
				f0f15f24 := &svcsdktypes.VCpuCountRangeRequest{}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.VCPUCount.Max != nil {
					maxCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.VCPUCount.Max
					if maxCopy0 > math.MaxInt32 || maxCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Max is of type int32")
					}
					maxCopy := int32(maxCopy0)
					f0f15f24.Max = &maxCopy
				}
				if r.ko.Spec.LaunchTemplateData.InstanceRequirements.VCPUCount.Min != nil {
					minCopy0 := *r.ko.Spec.LaunchTemplateData.InstanceRequirements.VCPUCount.Min
					if minCopy0 > math.MaxInt32 || minCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Min is of type int32")
					}
					minCopy := int32(minCopy0)
					f0f15f24.Min = &minCopy
				}
				f0f15.VCpuCount = f0f15f24
			}
			f0.InstanceRequirements = f0f15
		}
		if r.ko.Spec.LaunchTemplateData.InstanceType != nil {
			f0.InstanceType = svcsdktypes.InstanceType(*r.ko.Spec.LaunchTemplateData.InstanceType)
		}
		if r.ko.Spec.LaunchTemplateData.KernelID != nil {
			f0.KernelId = r.ko.Spec.LaunchTemplateData.KernelID
		}
		if r.ko.Spec.LaunchTemplateData.KeyName != nil {
			f0.KeyName = r.ko.Spec.LaunchTemplateData.KeyName
		}
		if r.ko.Spec.LaunchTemplateData.LicenseSpecifications != nil {
			f0f19 := []svcsdktypes.LaunchTemplateLicenseConfigurationRequest{}
			for _, f0f19iter := range r.ko.Spec.LaunchTemplateData.LicenseSpecifications {
				f0f19elem := &svcsdktypes.LaunchTemplateLicenseConfigurationRequest{}
				if f0f19iter.LicenseConfigurationARN != nil {
					f0f19elem.LicenseConfigurationArn = f0f19iter.LicenseConfigurationARN
				}
				f0f19 = append(f0f19, *f0f19elem)
			}
			f0.LicenseSpecifications = f0f19
		}
		if r.ko.Spec.LaunchTemplateData.MaintenanceOptions != nil {
			f0f20 := &svcsdktypes.LaunchTemplateInstanceMaintenanceOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.MaintenanceOptions.AutoRecovery != nil {
				f0f20.AutoRecovery = svcsdktypes.LaunchTemplateAutoRecoveryState(*r.ko.Spec.LaunchTemplateData.MaintenanceOptions.AutoRecovery)
			}
			f0.MaintenanceOptions = f0f20
		}
		if r.ko.Spec.LaunchTemplateData.MetadataOptions != nil {
			f0f21 := &svcsdktypes.LaunchTemplateInstanceMetadataOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPEndpoint != nil {
				f0f21.HttpEndpoint = svcsdktypes.LaunchTemplateInstanceMetadataEndpointState(*r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPEndpoint)
			}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPProtocolIPv6 != nil {
				f0f21.HttpProtocolIpv6 = svcsdktypes.LaunchTemplateInstanceMetadataProtocolIpv6(*r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPProtocolIPv6)
			}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPPutResponseHopLimit != nil {
				httpPutResponseHopLimitCopy0 := *r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPPutResponseHopLimit
				if httpPutResponseHopLimitCopy0 > math.MaxInt32 || httpPutResponseHopLimitCopy0 < math.MinInt32 {
					return fmt.Errorf("error: field HttpPutResponseHopLimit is of type int32")
				}
				httpPutResponseHopLimitCopy := int32(httpPutResponseHopLimitCopy0)
				f0f21.HttpPutResponseHopLimit = &httpPutResponseHopLimitCopy
			}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPTokens != nil {
				f0f21.HttpTokens = svcsdktypes.LaunchTemplateHttpTokensState(*r.ko.Spec.LaunchTemplateData.MetadataOptions.HTTPTokens)
			}
			if r.ko.Spec.LaunchTemplateData.MetadataOptions.InstanceMetadataTags != nil {
				f0f21.InstanceMetadataTags = svcsdktypes.LaunchTemplateInstanceMetadataTagsState(*r.ko.Spec.LaunchTemplateData.MetadataOptions.InstanceMetadataTags)
			}
			f0.MetadataOptions = f0f21
		}
		if r.ko.Spec.LaunchTemplateData.Monitoring != nil {
			f0f22 := &svcsdktypes.LaunchTemplatesMonitoringRequest{}
			if r.ko.Spec.LaunchTemplateData.Monitoring.Enabled != nil {
				f0f22.Enabled = r.ko.Spec.LaunchTemplateData.Monitoring.Enabled
			}
			f0.Monitoring = f0f22
		}
		if r.ko.Spec.LaunchTemplateData.NetworkInterfaces != nil {
			f0f23 := []svcsdktypes.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{}
			for _, f0f23iter := range r.ko.Spec.LaunchTemplateData.NetworkInterfaces {
				f0f23elem := &svcsdktypes.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{}
				if f0f23iter.AssociateCarrierIPAddress != nil {
					f0f23elem.AssociateCarrierIpAddress = f0f23iter.AssociateCarrierIPAddress
				}
				if f0f23iter.AssociatePublicIPAddress != nil {
					f0f23elem.AssociatePublicIpAddress = f0f23iter.AssociatePublicIPAddress
				}
				if f0f23iter.DeleteOnTermination != nil {
					f0f23elem.DeleteOnTermination = f0f23iter.DeleteOnTermination
				}
				if f0f23iter.Description != nil {
					f0f23elem.Description = f0f23iter.Description
				}
				if f0f23iter.DeviceIndex != nil {
					deviceIndexCopy0 := *f0f23iter.DeviceIndex
					if deviceIndexCopy0 > math.MaxInt32 || deviceIndexCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field DeviceIndex is of type int32")
					}
					deviceIndexCopy := int32(deviceIndexCopy0)
					f0f23elem.DeviceIndex = &deviceIndexCopy
				}
				if f0f23iter.Groups != nil {
					f0f23elem.Groups = aws.ToStringSlice(f0f23iter.Groups)
				}
				if f0f23iter.InterfaceType != nil {
					f0f23elem.InterfaceType = f0f23iter.InterfaceType
				}
				if f0f23iter.IPv4PrefixCount != nil {
					ipv4PrefixCountCopy0 := *f0f23iter.IPv4PrefixCount
					if ipv4PrefixCountCopy0 > math.MaxInt32 || ipv4PrefixCountCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Ipv4PrefixCount is of type int32")
					}
					ipv4PrefixCountCopy := int32(ipv4PrefixCountCopy0)
					f0f23elem.Ipv4PrefixCount = &ipv4PrefixCountCopy
				}
				if f0f23iter.IPv4Prefixes != nil {
					f0f23elemf8 := []svcsdktypes.Ipv4PrefixSpecificationRequest{}
					for _, f0f23elemf8iter := range f0f23iter.IPv4Prefixes {
						f0f23elemf8elem := &svcsdktypes.Ipv4PrefixSpecificationRequest{}
						if f0f23elemf8iter.IPv4Prefix != nil {
							f0f23elemf8elem.Ipv4Prefix = f0f23elemf8iter.IPv4Prefix
						}
						f0f23elemf8 = append(f0f23elemf8, *f0f23elemf8elem)
					}
					f0f23elem.Ipv4Prefixes = f0f23elemf8
				}
				if f0f23iter.IPv6AddressCount != nil {
					ipv6AddressCountCopy0 := *f0f23iter.IPv6AddressCount
					if ipv6AddressCountCopy0 > math.MaxInt32 || ipv6AddressCountCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Ipv6AddressCount is of type int32")
					}
					ipv6AddressCountCopy := int32(ipv6AddressCountCopy0)
					f0f23elem.Ipv6AddressCount = &ipv6AddressCountCopy
				}
				if f0f23iter.IPv6Addresses != nil {
					f0f23elemf10 := []svcsdktypes.InstanceIpv6AddressRequest{}
					for _, f0f23elemf10iter := range f0f23iter.IPv6Addresses {
						f0f23elemf10elem := &svcsdktypes.InstanceIpv6AddressRequest{}
						if f0f23elemf10iter.IPv6Address != nil {
							f0f23elemf10elem.Ipv6Address = f0f23elemf10iter.IPv6Address
						}
						f0f23elemf10 = append(f0f23elemf10, *f0f23elemf10elem)
					}
					f0f23elem.Ipv6Addresses = f0f23elemf10
				}
				if f0f23iter.IPv6PrefixCount != nil {
					ipv6PrefixCountCopy0 := *f0f23iter.IPv6PrefixCount
					if ipv6PrefixCountCopy0 > math.MaxInt32 || ipv6PrefixCountCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field Ipv6PrefixCount is of type int32")
					}
					ipv6PrefixCountCopy := int32(ipv6PrefixCountCopy0)
					f0f23elem.Ipv6PrefixCount = &ipv6PrefixCountCopy
				}
				if f0f23iter.IPv6Prefixes != nil {
					f0f23elemf12 := []svcsdktypes.Ipv6PrefixSpecificationRequest{}
					for _, f0f23elemf12iter := range f0f23iter.IPv6Prefixes {
						f0f23elemf12elem := &svcsdktypes.Ipv6PrefixSpecificationRequest{}
						if f0f23elemf12iter.IPv6Prefix != nil {
							f0f23elemf12elem.Ipv6Prefix = f0f23elemf12iter.IPv6Prefix
						}
						f0f23elemf12 = append(f0f23elemf12, *f0f23elemf12elem)
					}
					f0f23elem.Ipv6Prefixes = f0f23elemf12
				}
				if f0f23iter.NetworkCardIndex != nil {
					networkCardIndexCopy0 := *f0f23iter.NetworkCardIndex
					if networkCardIndexCopy0 > math.MaxInt32 || networkCardIndexCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field NetworkCardIndex is of type int32")
					}
					networkCardIndexCopy := int32(networkCardIndexCopy0)
					f0f23elem.NetworkCardIndex = &networkCardIndexCopy
				}
				if f0f23iter.NetworkInterfaceID != nil {
					f0f23elem.NetworkInterfaceId = f0f23iter.NetworkInterfaceID
				}
				if f0f23iter.PrimaryIPv6 != nil {
					f0f23elem.PrimaryIpv6 = f0f23iter.PrimaryIPv6
				}
				if f0f23iter.PrivateIPAddress != nil {
					f0f23elem.PrivateIpAddress = f0f23iter.PrivateIPAddress
				}
				if f0f23iter.PrivateIPAddresses != nil {
					f0f23elemf17 := []svcsdktypes.PrivateIpAddressSpecification{}
					for _, f0f23elemf17iter := range f0f23iter.PrivateIPAddresses {
						f0f23elemf17elem := &svcsdktypes.PrivateIpAddressSpecification{}
						if f0f23elemf17iter.Primary != nil {
							f0f23elemf17elem.Primary = f0f23elemf17iter.Primary
						}
						if f0f23elemf17iter.PrivateIPAddress != nil {
							f0f23elemf17elem.PrivateIpAddress = f0f23elemf17iter.PrivateIPAddress
						}
						f0f23elemf17 = append(f0f23elemf17, *f0f23elemf17elem)
					}
					f0f23elem.PrivateIpAddresses = f0f23elemf17
				}
				if f0f23iter.SecondaryPrivateIPAddressCount != nil {
					secondaryPrivateIPAddressCountCopy0 := *f0f23iter.SecondaryPrivateIPAddressCount
					if secondaryPrivateIPAddressCountCopy0 > math.MaxInt32 || secondaryPrivateIPAddressCountCopy0 < math.MinInt32 {
						return fmt.Errorf("error: field SecondaryPrivateIpAddressCount is of type int32")
					}
					secondaryPrivateIPAddressCountCopy := int32(secondaryPrivateIPAddressCountCopy0)
					f0f23elem.SecondaryPrivateIpAddressCount = &secondaryPrivateIPAddressCountCopy
				}
				if f0f23iter.SubnetID != nil {
					f0f23elem.SubnetId = f0f23iter.SubnetID
				}
				f0f23 = append(f0f23, *f0f23elem)
			}
			f0.NetworkInterfaces = f0f23
		}
		if r.ko.Spec.LaunchTemplateData.Placement != nil {
			f0f24 := &svcsdktypes.LaunchTemplatePlacementRequest{}
			if r.ko.Spec.LaunchTemplateData.Placement.Affinity != nil {
				f0f24.Affinity = r.ko.Spec.LaunchTemplateData.Placement.Affinity
			}
			if r.ko.Spec.LaunchTemplateData.Placement.AvailabilityZone != nil {
				f0f24.AvailabilityZone = r.ko.Spec.LaunchTemplateData.Placement.AvailabilityZone
			}
			if r.ko.Spec.LaunchTemplateData.Placement.GroupID != nil {
				f0f24.GroupId = r.ko.Spec.LaunchTemplateData.Placement.GroupID
			}
			if r.ko.Spec.LaunchTemplateData.Placement.GroupName != nil {
				f0f24.GroupName = r.ko.Spec.LaunchTemplateData.Placement.GroupName
			}
			if r.ko.Spec.LaunchTemplateData.Placement.HostID != nil {
				f0f24.HostId = r.ko.Spec.LaunchTemplateData.Placement.HostID
			}
			if r.ko.Spec.LaunchTemplateData.Placement.HostResourceGroupARN != nil {
				f0f24.HostResourceGroupArn = r.ko.Spec.LaunchTemplateData.Placement.HostResourceGroupARN
			}
			if r.ko.Spec.LaunchTemplateData.Placement.PartitionNumber != nil {
				partitionNumberCopy0 := *r.ko.Spec.LaunchTemplateData.Placement.PartitionNumber
				if partitionNumberCopy0 > math.MaxInt32 || partitionNumberCopy0 < math.MinInt32 {
					return fmt.Errorf("error: field PartitionNumber is of type int32")
				}
				partitionNumberCopy := int32(partitionNumberCopy0)
				f0f24.PartitionNumber = &partitionNumberCopy
			}
			if r.ko.Spec.LaunchTemplateData.Placement.SpreadDomain != nil {
				f0f24.SpreadDomain = r.ko.Spec.LaunchTemplateData.Placement.SpreadDomain
			}
			if r.ko.Spec.LaunchTemplateData.Placement.Tenancy != nil {
				f0f24.Tenancy = svcsdktypes.Tenancy(*r.ko.Spec.LaunchTemplateData.Placement.Tenancy)
			}
			f0.Placement = f0f24
		}
		if r.ko.Spec.LaunchTemplateData.PrivateDNSNameOptions != nil {
			f0f25 := &svcsdktypes.LaunchTemplatePrivateDnsNameOptionsRequest{}
			if r.ko.Spec.LaunchTemplateData.PrivateDNSNameOptions.EnableResourceNameDNSAAAARecord != nil {
				f0f25.EnableResourceNameDnsAAAARecord = r.ko.Spec.LaunchTemplateData.PrivateDNSNameOptions.EnableResourceNameDNSAAAARecord
			}
			if r.ko.Spec.LaunchTemplateData.PrivateDNSNameOptions.EnableResourceNameDNSARecord != nil {
				f0f25.EnableResourceNameDnsARecord = r.ko.Spec.LaunchTemplateData.PrivateDNSNameOptions.EnableResourceNameDNSARecord
			}
			if r.ko.Spec.LaunchTemplateData.PrivateDNSNameOptions.HostnameType != nil {
				f0f25.HostnameType = svcsdktypes.HostnameType(*r.ko.Spec.LaunchTemplateData.PrivateDNSNameOptions.HostnameType)
			}
			f0.PrivateDnsNameOptions = f0f25
		}
		if r.ko.Spec.LaunchTemplateData.RAMDiskID != nil {
			f0.RamDiskId = r.ko.Spec.LaunchTemplateData.RAMDiskID
		}
		if r.ko.Spec.LaunchTemplateData.SecurityGroupIDs != nil {
			f0.SecurityGroupIds = aws.ToStringSlice(r.ko.Spec.LaunchTemplateData.SecurityGroupIDs)
		}
		if r.ko.Spec.LaunchTemplateData.SecurityGroups != nil {
			f0.SecurityGroups = aws.ToStringSlice(r.ko.Spec.LaunchTemplateData.SecurityGroups)
		}
		if r.ko.Spec.LaunchTemplateData.TagSpecifications != nil {
			f0f29 := []svcsdktypes.LaunchTemplateTagSpecificationRequest{}
			for _, f0f29iter := range r.ko.Spec.LaunchTemplateData.TagSpecifications {
				f0f29elem := &svcsdktypes.LaunchTemplateTagSpecificationRequest{}
				if f0f29iter.ResourceType != nil {
					f0f29elem.ResourceType = svcsdktypes.ResourceType(*f0f29iter.ResourceType)
				}
				if f0f29iter.Tags != nil {
					f0f29elemf1 := []svcsdktypes.Tag{}
					for _, f0f29elemf1iter := range f0f29iter.Tags {
						f0f29elemf1elem := &svcsdktypes.Tag{}
						if f0f29elemf1iter.Key != nil {
							f0f29elemf1elem.Key = f0f29elemf1iter.Key
						}
						if f0f29elemf1iter.Value != nil {
							f0f29elemf1elem.Value = f0f29elemf1iter.Value
						}
						f0f29elemf1 = append(f0f29elemf1, *f0f29elemf1elem)
					}
					f0f29elem.Tags = f0f29elemf1
				}
				f0f29 = append(f0f29, *f0f29elem)
			}
			f0.TagSpecifications = f0f29
		}
		if r.ko.Spec.LaunchTemplateData.UserData != nil {
			f0.UserData = r.ko.Spec.LaunchTemplateData.UserData
		}
		input.LaunchTemplateData = f0
	}

	// create newlaunchtemplateversion
	_, err = rm.sdkapi.CreateLaunchTemplateVersion(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateLaunchTemplateVersion", err)
	if err != nil {
		return err
	}

	return nil
}
