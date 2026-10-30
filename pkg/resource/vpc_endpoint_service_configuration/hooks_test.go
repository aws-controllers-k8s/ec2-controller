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

package vpc_endpoint_service_configuration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

func TestSdkDelete_NotDeleted(t *testing.T) {
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ = url.ParseQuery(string(body))
		_, _ = io.WriteString(w, `<DeleteVpcEndpointServiceConfigurationsResponse><unsuccessful><item><error><code>OperationNotPermitted</code><message>The service configuration cannot be deleted.</message></error><resourceId>vpce-svc-0123</resourceId></item></unsuccessful></DeleteVpcEndpointServiceConfigurationsResponse>`)
	}))
	t.Cleanup(srv.Close)
	rm := &resourceManager{
		sdkapi:  svcsdk.New(svcsdk.Options{Region: "us-west-2", BaseEndpoint: aws.String(srv.URL), Credentials: aws.AnonymousCredentials{}}),
		metrics: ackmetrics.NewMetrics("ec2"),
	}
	ko := &svcapitypes.VPCEndpointServiceConfiguration{}
	ko.Status.ServiceID = aws.String("vpce-svc-0123")

	_, err := rm.sdkDelete(context.Background(), &resource{ko})

	assert.Equal(t, "vpce-svc-0123", form.Get("ServiceId.1"))
	assert.EqualError(t, err, "api error OperationNotPermitted: The service configuration cannot be deleted.")
}
