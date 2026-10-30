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

package flow_log

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	smithy "github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

// stubEC2 returns a resource manager whose EC2 client posts to a stub that
// answers every call with response, and the form of the last call.
func stubEC2(t *testing.T, response string) (*resourceManager, *url.Values) {
	form := &url.Values{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*form, _ = url.ParseQuery(string(body))
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)
	return &resourceManager{
		sdkapi:  svcsdk.New(svcsdk.Options{Region: "us-west-2", BaseEndpoint: aws.String(srv.URL), Credentials: aws.AnonymousCredentials{}}),
		metrics: ackmetrics.NewMetrics("ec2"),
	}, form
}

func TestSdkDelete(t *testing.T) {
	for _, tc := range []struct {
		name         string
		unsuccessful string
		wantErr      bool
		wantCode     string
		wantMessage  string
	}{
		{
			name: "EC2 deletes the flow log",
		},
		{
			name:         "EC2 no longer has the flow log",
			unsuccessful: `<item><error><code>InvalidFlowLogId.NotFound</code><message>The flow log does not exist.</message></error><resourceId>fl-123</resourceId></item>`,
		},
		{
			name:         "EC2 does not delete the flow log",
			unsuccessful: `<item><error><code>UnauthorizedOperation</code><message>The flow log cannot be deleted.</message></error><resourceId>fl-123</resourceId></item>`,
			wantErr:      true,
			wantCode:     "UnauthorizedOperation",
			wantMessage:  "The flow log cannot be deleted.",
		},
		{
			name:         "EC2 lists the flow log without an error",
			unsuccessful: `<item><resourceId>fl-123</resourceId></item>`,
			wantErr:      true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rm, form := stubEC2(t, `<DeleteFlowLogsResponse><unsuccessful>`+tc.unsuccessful+`</unsuccessful></DeleteFlowLogsResponse>`)
			ko := &svcapitypes.FlowLog{}
			ko.Status.FlowLogID = aws.String("fl-123")

			_, err := rm.sdkDelete(context.Background(), &resource{ko})

			assert.Equal(t, "fl-123", form.Get("FlowLogId.1"))
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tc.wantCode == "" {
				return
			}
			var apiErr smithy.APIError
			require.True(t, errors.As(err, &apiErr))
			assert.Equal(t, tc.wantCode, apiErr.ErrorCode())
			assert.Equal(t, tc.wantMessage, apiErr.ErrorMessage())
		})
	}
}

func desiredFlowLog() *resource {
	ko := &svcapitypes.FlowLog{}
	ko.Spec.ResourceID = aws.String("vpc-0123")
	ko.Spec.ResourceType = aws.String("VPC")
	return &resource{ko}
}

func TestSdkCreate_Unsuccessful(t *testing.T) {
	rm, _ := stubEC2(t, `<CreateFlowLogsResponse><flowLogIdSet/><unsuccessful><item><error><code>InvalidParameter</code><message>The log destination is invalid.</message></error><resourceId>vpc-0123</resourceId></item></unsuccessful></CreateFlowLogsResponse>`)

	created, err := rm.sdkCreate(context.Background(), desiredFlowLog())

	assert.Nil(t, created)
	require.Error(t, err)
	assert.False(t, ackerr.IsPostCreateError(err))
	var apiErr smithy.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "InvalidParameter", apiErr.ErrorCode())
	assert.Equal(t, "The log destination is invalid.", apiErr.ErrorMessage())
}

func TestSdkCreate_Success(t *testing.T) {
	rm, form := stubEC2(t, `<CreateFlowLogsResponse><flowLogIdSet><item>fl-123</item></flowLogIdSet><unsuccessful/></CreateFlowLogsResponse>`)

	created, err := rm.sdkCreate(context.Background(), desiredFlowLog())

	require.NoError(t, err)
	assert.Equal(t, "CreateFlowLogs", form.Get("Action"))
	require.NotNil(t, created)
	assert.Equal(t, "fl-123", aws.ToString(created.ko.Status.FlowLogID))
}
