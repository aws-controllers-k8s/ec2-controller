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

package reservation

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// addInstanceIDsToTerminateRequest uses Reservation InstanceIDs to populate instances field in
// a TerminateInstances request.
// Return error to indicate to callers that the resource is not yet created.
func addInstanceIDsToTerminateRequest(r *resource,
	input *svcsdk.TerminateInstancesInput) error {
	if r.ko.Status.Instances == nil || len(r.ko.Status.Instances) <= 0 {
		return errors.New("unable to extract InstanceIDs from Reservation")
	}
	for _, instance := range r.ko.Status.Instances {
		input.InstanceIds = append(input.InstanceIds, instance.InstanceID)
	}
	return nil
}

// addReservationIDToListRequest populates the Filter in a DescribeInstances request
// with the ReservationID
// Return error to indicate to callers that the resource is not yet created.
func addReservationIDToListRequest(r *resource,
	input *svcsdk.DescribeInstancesInput) error {
	if r.ko.Status.ReservationID == nil {
		return errors.New("unable to extract ReservationID from Reservation")
	}
	input.Filters = []*svcsdk.Filter{
		{
			Name:   aws.String("reservation-id"),
			Values: []*string{r.ko.Status.ReservationID},
		},
	}
	return nil
}
