package transit_gateway_vpc_attachment

import (
	"errors"
	"testing"

	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

func TestRequeueWaitUntilCanModifyAlwaysResolvesToRequeueAfter(t *testing.T) {
	states := map[string]*string{
		"pending":     aws.String("pending"),
		"modifying":   aws.String("modifying"),
		"failing":     aws.String("failing"),
		"unset state": nil,
	}

	for name, state := range states {
		t.Run(name, func(t *testing.T) {
			r := &resource{
				ko: &svcapitypes.TransitGatewayVPCAttachment{
					Status: svcapitypes.TransitGatewayVPCAttachmentStatus{
						State: state,
					},
				},
			}

			var err error = requeueWaitUntilCanModify(r)
			require.NotNil(t, err)

			var after *ackrequeue.RequeueNeededAfter
			require.True(t, errors.As(err, &after))
			assert.Positive(t, after.Duration())
			assert.NotEmpty(t, err.Error())
		})
	}
}
