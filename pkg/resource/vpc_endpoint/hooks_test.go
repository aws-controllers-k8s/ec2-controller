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

package vpc_endpoint

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

func endpoint(specIDs []*string, awsIDs ...string) *resource {
	ko := &svcapitypes.VPCEndpoint{}
	ko.Spec.SecurityGroupIDs = specIDs
	ko.Status.VPCEndpointID = aws.String("vpce-0123")
	for _, id := range awsIDs {
		ko.Status.Groups = append(ko.Status.Groups, &svcapitypes.SecurityGroupIdentifier{GroupID: aws.String(id)})
	}
	return &resource{ko}
}

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

// AWS reports sg-1 on the endpoint and the spec asks for sg-1 and sg-2, so
// the resource sdkFind returns lists sg-1 in its spec and differs from the
// desired one.
func TestSdkFind_MirrorsSecurityGroupIDs(t *testing.T) {
	rm, _ := stubEC2(t, `<DescribeVpcEndpointsResponse><vpcEndpointSet><item><vpcEndpointId>vpce-0123</vpcEndpointId><state>available</state><groupSet><item><groupId>sg-1</groupId></item></groupSet></item></vpcEndpointSet></DescribeVpcEndpointsResponse>`)
	desired := endpoint(aws.StringSlice([]string{"sg-1", "sg-2"}))

	latest, err := rm.sdkFind(context.Background(), desired)
	require.NoError(t, err)

	assert.Equal(t, aws.StringSlice([]string{"sg-1"}), latest.ko.Spec.SecurityGroupIDs)
	assert.True(t, newResourceDelta(desired, latest).DifferentAt("Spec.SecurityGroupIDs"))
}

// sdkFind starts from a copy of the desired resource, so latest below is the
// desired spec with the groups AWS reports in Status.Groups.
func TestMirrorSecurityGroupIDs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		spec      []*string
		aws       []string
		wantSpec  []*string
		wantDelta bool
	}{
		{name: "a group added to the spec", spec: aws.StringSlice([]string{"sg-1", "sg-2"}), aws: []string{"sg-1"}, wantSpec: aws.StringSlice([]string{"sg-1"}), wantDelta: true},
		{name: "a group removed from the spec", spec: aws.StringSlice([]string{"sg-1"}), aws: []string{"sg-1", "sg-2"}, wantSpec: aws.StringSlice([]string{"sg-1", "sg-2"}), wantDelta: true},
		{name: "AWS reports no groups", spec: aws.StringSlice([]string{"sg-1"}), wantSpec: []*string{}, wantDelta: true},
		{name: "the same groups in another order", spec: aws.StringSlice([]string{"sg-2", "sg-1"}), aws: []string{"sg-1", "sg-2"}, wantSpec: aws.StringSlice([]string{"sg-1", "sg-2"})},
		{name: "a spec with no groups keeps the endpoint's groups", aws: []string{"sg-default"}},
		{name: "a spec with an empty list keeps the endpoint's groups", spec: []*string{}, aws: []string{"sg-default"}, wantSpec: []*string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired := endpoint(tc.spec)
			latest := endpoint(tc.spec, tc.aws...)
			latest.ko.Status.Groups = append(latest.ko.Status.Groups, &svcapitypes.SecurityGroupIdentifier{GroupName: aws.String("no-id")})
			mirrorSecurityGroupIDs(latest.ko)

			assert.Equal(t, tc.wantSpec, latest.ko.Spec.SecurityGroupIDs)
			assert.Equal(t, tc.wantDelta, newResourceDelta(desired, latest).DifferentAt("Spec.SecurityGroupIDs"))
		})
	}
}

func TestCustomUpdateVPCEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name        string
		desired     *resource
		latest      *resource
		wantForm    url.Values
		wantChanged bool
	}{
		{
			name:        "a changed group list adds and removes groups and reads the endpoint again soon",
			desired:     endpoint(aws.StringSlice([]string{"sg-1", "sg-2"})),
			latest:      endpoint(aws.StringSlice([]string{"sg-1", "sg-3"})),
			wantForm:    url.Values{"AddSecurityGroupId.1": {"sg-2"}, "RemoveSecurityGroupId.1": {"sg-3"}},
			wantChanged: true,
		},
		{
			name:        "an added group is added and none is removed",
			desired:     endpoint(aws.StringSlice([]string{"sg-1", "sg-2"})),
			latest:      endpoint(aws.StringSlice([]string{"sg-1"})),
			wantForm:    url.Values{"AddSecurityGroupId.1": {"sg-2"}, "RemoveSecurityGroupId.1": nil},
			wantChanged: true,
		},
		{
			name:     "another change leaves the synced condition to the runtime",
			desired:  &resource{&svcapitypes.VPCEndpoint{Spec: svcapitypes.VPCEndpointSpec{PrivateDNSEnabled: aws.Bool(true)}, Status: svcapitypes.VPCEndpointStatus{VPCEndpointID: aws.String("vpce-0123")}}},
			latest:   &resource{&svcapitypes.VPCEndpoint{Spec: svcapitypes.VPCEndpointSpec{PrivateDNSEnabled: aws.Bool(false)}, Status: svcapitypes.VPCEndpointStatus{VPCEndpointID: aws.String("vpce-0123")}}},
			wantForm: url.Values{"PrivateDnsEnabled": {"true"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rm, form := stubEC2(t, `<ModifyVpcEndpointResponse><return>true</return></ModifyVpcEndpointResponse>`)

			updated, err := rm.customUpdateVPCEndpoint(context.Background(), tc.desired, tc.latest, newResourceDelta(tc.desired, tc.latest))
			require.NoError(t, err)

			assert.Equal(t, "ModifyVpcEndpoint", form.Get("Action"))
			for key, want := range tc.wantForm {
				assert.Equal(t, want, (*form)[key], key)
			}
			synced := ackcondition.Synced(updated)
			if !tc.wantChanged {
				assert.Nil(t, synced)
				return
			}
			require.NotNil(t, synced)
			assert.Equal(t, corev1.ConditionFalse, synced.Status)
			assert.Equal(t, ackv1alpha1.ConditionTypeResourceSynced, synced.Type)
			assert.NotNil(t, synced.Message)
		})
	}
}

func TestSdkDelete(t *testing.T) {
	for _, tc := range []struct {
		name         string
		unsuccessful string
		wantErr      string
	}{
		{
			name: "EC2 deletes the endpoint",
		},
		{
			name:         "EC2 does not delete the endpoint",
			unsuccessful: `<item><error><code>OperationNotPermitted</code><message>The endpoint cannot be deleted.</message></error><resourceId>vpce-0123</resourceId></item>`,
			wantErr:      "failed to delete VPC Endpoint: The endpoint cannot be deleted.",
		},
		{
			name:         "EC2 no longer has the endpoint",
			unsuccessful: `<item><error><code>InvalidVpcEndpoint.NotFound</code><message>The VPC endpoint does not exist.</message></error><resourceId>vpce-0123</resourceId></item>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rm, form := stubEC2(t, `<DeleteVpcEndpointsResponse><unsuccessful>`+tc.unsuccessful+`</unsuccessful></DeleteVpcEndpointsResponse>`)

			_, err := rm.sdkDelete(context.Background(), endpoint(nil))

			assert.Equal(t, "vpce-0123", form.Get("VpcEndpointId.1"))
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}
