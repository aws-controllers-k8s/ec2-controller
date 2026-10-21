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

package key_pair

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

const testPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPfh4lRy/xJUbMAlfUO6qYyTTnKLDwducZGNA7VIgbz5 test"

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

func TestSdkCreate_ImportsPublicKeyWithTags(t *testing.T) {
	rm, form := stubEC2(t, `<ImportKeyPairResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><requestId>req</requestId><keyName>test-key</keyName><keyFingerprint>SHA256:abc</keyFingerprint><keyPairId>key-0123456789abcdef0</keyPairId></ImportKeyPairResponse>`)
	desired := &resource{ko: &svcapitypes.KeyPair{}}
	desired.ko.Spec.KeyName = aws.String("test-key")
	desired.ko.Spec.PublicKeyMaterial = aws.String(testPublicKey)
	desired.ko.Spec.Tags = []*svcapitypes.Tag{{Key: aws.String("env"), Value: aws.String("test")}}

	created, err := rm.sdkCreate(context.Background(), desired)
	if err != nil {
		t.Fatalf("sdkCreate: %v", err)
	}

	if got := form.Get("Action"); got != "ImportKeyPair" {
		t.Errorf("Action = %q, want ImportKeyPair", got)
	}
	if got := form.Get("PublicKeyMaterial"); got != base64.StdEncoding.EncodeToString([]byte(testPublicKey)) {
		t.Errorf("PublicKeyMaterial = %q, want the base64 of the public key text", got)
	}
	if got := form.Get("TagSpecification.1.ResourceType"); got != "key-pair" {
		t.Errorf("TagSpecification.1.ResourceType = %q, want key-pair", got)
	}
	if got := form.Get("TagSpecification.1.Tag.1.Key"); got != "env" {
		t.Errorf("TagSpecification.1.Tag.1.Key = %q, want env", got)
	}
	if got := aws.ToString(created.ko.Status.KeyPairID); got != "key-0123456789abcdef0" {
		t.Errorf("Status.KeyPairID = %q", got)
	}
	if got := aws.ToString(created.ko.Status.KeyFingerprint); got != "SHA256:abc" {
		t.Errorf("Status.KeyFingerprint = %q", got)
	}
	if got := aws.ToString(created.ko.Spec.PublicKeyMaterial); got != testPublicKey {
		t.Errorf("the public key text must stay in the spec, got %q", got)
	}
}

func TestSdkFind_ListsByNameAndIncludesPublicKey(t *testing.T) {
	rm, form := stubEC2(t, `<DescribeKeyPairsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><requestId>req</requestId><keySet><item><keyName>test-key</keyName><keyPairId>key-0123456789abcdef0</keyPairId><keyFingerprint>SHA256:abc</keyFingerprint><keyType>ed25519</keyType><publicKey>`+testPublicKey+`</publicKey></item></keySet></DescribeKeyPairsResponse>`)
	desired := &resource{ko: &svcapitypes.KeyPair{}}
	desired.ko.Spec.KeyName = aws.String("test-key")
	desired.ko.Spec.PublicKeyMaterial = aws.String(testPublicKey)

	latest, err := rm.sdkFind(context.Background(), desired)
	if err != nil {
		t.Fatalf("sdkFind: %v", err)
	}
	if got := form.Get("KeyName.1"); got != "test-key" {
		t.Errorf("KeyName.1 = %q, want test-key", got)
	}
	if got := form.Get("IncludePublicKey"); got != "true" {
		t.Errorf("IncludePublicKey = %q, want true", got)
	}
	if got := aws.ToString(latest.ko.Status.KeyPairID); got != "key-0123456789abcdef0" {
		t.Errorf("Status.KeyPairID = %q", got)
	}
	if newResourceDelta(desired, latest).DifferentAt("Spec") {
		t.Errorf("a key pair read back by name must not differ from its spec: %v", newResourceDelta(desired, latest).Differences)
	}
}

func TestUpdateTagSpecificationsInCreateRequest_NoTags(t *testing.T) {
	r := &resource{ko: &svcapitypes.KeyPair{}}
	input := &svcsdk.ImportKeyPairInput{}
	updateTagSpecificationsInCreateRequest(r, input)
	if input.TagSpecifications != nil {
		t.Errorf("no tags in the spec must leave TagSpecifications nil, got %v", input.TagSpecifications)
	}
}
