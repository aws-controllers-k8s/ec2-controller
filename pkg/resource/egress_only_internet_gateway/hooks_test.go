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

package egress_only_internet_gateway

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

func TestCheckRequiredFieldsMissing(t *testing.T) {
	rm := &resourceManager{}

	tests := []struct {
		name     string
		statusID *string
		expected bool
	}{
		{
			name:     "returns true when Status.ID is nil (new resource)",
			statusID: nil,
			expected: true,
		},
		{
			name:     "returns false when Status.ID is set (existing resource)",
			statusID: aws.String("eigw-0123456789abcdef0"),
			expected: false,
		},
		{
			name:     "returns false when Status.ID is empty string",
			statusID: aws.String(""),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &resource{
				ko: &svcapitypes.EgressOnlyInternetGateway{
					Status: svcapitypes.EgressOnlyInternetGatewayStatus{
						ID: tc.statusID,
					},
				},
			}
			result := rm.checkRequiredFieldsMissing(r)
			assert.Equal(t, tc.expected, result)
		})
	}
}
