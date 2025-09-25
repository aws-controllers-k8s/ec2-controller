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

package instance

import (
	"context"
	"errors"
	"fmt"
	"time"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

const (
	requeueUntilReadyDuration = 10 * time.Second
)

// addInstanceIDsToTerminateRequest populates the list of InstanceIDs
// in the TerminateInstances request with the resource's InstanceID
// Return error to indicate to callers that the resource is not yet created.
func addInstanceIDsToTerminateRequest(r *resource,
	input *svcsdk.TerminateInstancesInput) error {
	if r.ko.Status.InstanceID == nil {
		return errors.New("InstanceID nil for resource when creating TerminateRequest")
	}
	input.InstanceIds = append(input.InstanceIds, *r.ko.Status.InstanceID)
	return nil
}

func (rm *resourceManager) customUpdateInstance(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateInstance")
	defer func() { exit(err) }()

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())
	updated.SetStatus(latest)

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.InstanceID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return updated, err
		}
	}

	if !delta.DifferentExcept("Spec.Tags") {
		return updated, nil
	}

	if !isRunning(updated.ko) {
		return updated, ackrequeue.NeededAfter(
			fmt.Errorf("requeuing until state is %s or %s", svcsdktypes.InstanceStateNameRunning, svcsdktypes.InstanceStateNameStopped),
			requeueUntilReadyDuration,
		)
	}

	err = rm.modifyInstanceAttributes(ctx, delta, desired, latest)
	if err != nil {
		return updated, err
	}

	return updated, nil
}

func (rm *resourceManager) modifyInstanceAttributes(ctx context.Context, delta *ackcompare.Delta, desired, latest *resource) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.modifyInstanceAttributes")
	defer func() { exit(err) }()
	input := &svcsdk.ModifyInstanceAttributeInput{
		InstanceId: latest.ko.Status.InstanceID,
	}
	// we can only update one attribute at a time
	if delta.DifferentAt("Spec.DisableAPITermination") {
		input.DisableApiTermination = &svcsdktypes.AttributeBooleanValue{Value: desired.ko.Spec.DisableAPITermination}
	} else if delta.DifferentAt("Spec.InstanceType") {
		input.InstanceType = &svcsdktypes.AttributeValue{Value: desired.ko.Spec.InstanceType}
	} else if delta.DifferentAt("Spec.KernelID") {
		input.Kernel = &svcsdktypes.AttributeValue{Value: desired.ko.Spec.KernelID}
	} else if delta.DifferentAt("Spec.RAMDiskID") {
		input.Ramdisk = &svcsdktypes.AttributeValue{Value: desired.ko.Spec.RAMDiskID}
	} else if delta.DifferentAt("Spec.InstanceInitiatedShutdownBehavior") {
		input.InstanceInitiatedShutdownBehavior = &svcsdktypes.AttributeValue{Value: desired.ko.Spec.InstanceInitiatedShutdownBehavior}
	} else if delta.DifferentAt("Spec.UserData") {
		input.UserData = &svcsdktypes.BlobAttributeValue{Value: []byte(aws.ToString(desired.ko.Spec.UserData))}
	} else if delta.DifferentAt("Spec.EBSOptimized") {
		input.EbsOptimized = &svcsdktypes.AttributeBooleanValue{Value: desired.ko.Spec.EBSOptimized}
	} else if delta.DifferentAt("Spec.DisableAPIStop") {
		input.DisableApiStop = &svcsdktypes.AttributeBooleanValue{Value: desired.ko.Spec.DisableAPIStop}
	} else if delta.DifferentAt("Spec.SecurityGroupIDs") {
		input.Groups = aws.ToStringSlice(desired.ko.Spec.SecurityGroupIDs)
	} else {
		input = nil
	}

	if input != nil {
		_, err = rm.sdkapi.ModifyInstanceAttribute(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "ModifyInstanceAttribute", err)
		if err != nil {
			return err
		}
		return fmt.Errorf("requeuing until all fields are updated")
	}
	return nil
}

func isRunning(ko *v1alpha1.Instance) bool {
	if ko.Status.State == nil || ko.Status.State.Name == nil {
		return false
	}

	// NOTE: (michaelhtm) We will count `stopped` as running for now.
	// TODO: expose annotation to allow users to start/stop instances
	return *ko.Status.State.Name == string(svcsdktypes.InstanceStateNameRunning) ||
		*ko.Status.State.Name == string(svcsdktypes.InstanceStateNameStopped)
}

// needsRestart checks if the Instance is terminated (deleted)
func needsRestart(ko *v1alpha1.Instance) bool {
	if ko.Status.State == nil || ko.Status.State.Name == nil {
		return false
	}

	return *ko.Status.State.Name == string(svcsdktypes.InstanceStateNameTerminated)
}


func setAdditionalFields(instance svcsdktypes.Instance, ko *v1alpha1.Instance) {
	ko.Spec.SecurityGroupIDs = []*string{}
	for _, group := range instance.SecurityGroups {
		ko.Spec.SecurityGroupIDs = append(ko.Spec.SecurityGroupIDs, group.GroupId)
	}

	if monitoring := instance.Monitoring; monitoring != nil {
		switch monitoring.State {
		case svcsdktypes.MonitoringStateDisabled, svcsdktypes.MonitoringStateDisabling:
			ko.Spec.Monitoring = &v1alpha1.RunInstancesMonitoringEnabled{Enabled: aws.Bool(false)}

		case svcsdktypes.MonitoringStateEnabled, svcsdktypes.MonitoringStatePending:
			ko.Spec.Monitoring = &v1alpha1.RunInstancesMonitoringEnabled{Enabled: aws.Bool(true)}
		}
	}
}

var computeTagsDelta = tags.ComputeTagsDelta

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to RunInstancesInput.TagSpecification
// and ensures the ResourceType is always set to 'instance'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.RunInstancesInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		instanceTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.Key = desiredTag.Key
				tag.Value = desiredTag.Value
			}
			instanceTags = append(instanceTags, tag)
		}
		desiredTagSpecs.ResourceType = "instance"
		desiredTagSpecs.Tags = instanceTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}
