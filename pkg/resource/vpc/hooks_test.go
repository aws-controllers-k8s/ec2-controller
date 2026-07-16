package vpc

import (
	"context"
	"testing"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

func TestCustomUpdateVPCPropagatesCIDRBlockStatusChanges(t *testing.T) {
	desired := &resource{
		ko: &svcapitypes.VPC{
			Spec: svcapitypes.VPCSpec{
				CIDRBlocks: []*string{aws.String("10.0.0.0/16")},
			},
		},
	}
	latest := &resource{
		ko: &svcapitypes.VPC{
			Spec: svcapitypes.VPCSpec{
				CIDRBlocks: []*string{aws.String("10.0.0.0/16")},
			},
			Status: svcapitypes.VPCStatus{
				CIDRBlockAssociationSet: []*svcapitypes.VPCCIDRBlockAssociation{
					{CIDRBlock: aws.String("10.0.0.0/16")},
				},
				IPv6CIDRBlockAssociationSet: []*svcapitypes.VPCIPv6CIDRBlockAssociation{
					{IPv6CIDRBlock: aws.String("2600:1f18::/56")},
				},
			},
		},
	}

	rm := &resourceManager{}
	delta := ackcompare.NewDelta()

	updated, err := rm.customUpdateVPC(context.Background(), desired, latest, delta)
	require.NoError(t, err)
	assert.Equal(t, latest.ko.Status.CIDRBlockAssociationSet, updated.ko.Status.CIDRBlockAssociationSet)
	assert.Equal(t, latest.ko.Status.IPv6CIDRBlockAssociationSet, updated.ko.Status.IPv6CIDRBlockAssociationSet)
}
