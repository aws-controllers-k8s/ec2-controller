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

package vpc

import (
	"errors"

	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// isRequiredFieldsMissingFromInput returns True if CR is
// missing required fields for reconciliation.
func isRequiredFieldsMissingFromInput(r *resource) bool {
	return r.ko.Status.State == nil
}

// addIDToListRequest adds requested-resource VpcId to ListRequest. Return error to indicate to callers that the
// resource is not yet created.
func addIDToListRequest(r *resource, input *svcsdk.DescribeVpcsInput) error {
	if r.ko.Status.VPCID == nil {
		return errors.New("unable to extract vpcId from Kubernetes resource")
	}
	input.VpcIds = []*string{r.ko.Status.VPCID}
	return nil
}
