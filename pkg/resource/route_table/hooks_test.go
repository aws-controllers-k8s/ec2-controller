package route_table

import (
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestCustomerPreCompare(t *testing.T) {
	type Routes []*svcapitypes.CreateRouteInput

	peeringRoute := func(pcxID string, cidr string) *svcapitypes.CreateRouteInput {
		return &svcapitypes.CreateRouteInput{
			DestinationCIDRBlock:   aws.String(cidr),
			VPCPeeringConnectionID: aws.String(pcxID),
		}
	}

	createRouteTableTestResource := func(routes []*svcapitypes.CreateRouteInput) *resource {
		return &resource{
			ko: &svcapitypes.RouteTable{
				Spec: svcapitypes.RouteTableSpec{
					Routes: routes,
				},
			},
		}
	}

	assertRoutesIdentical := func(t *testing.T, a, b []*svcapitypes.CreateRouteInput) {
		assert.Len(t, a, len(b))
		for i := range a {
			assert.EqualValues(t, a[i], b[i])
		}
	}

	tt := []struct {
		id            string
		desiredRoutes Routes
		latestRoutes  Routes
		toAdd         Routes
		toDelete      Routes
	}{
		{"all identical",
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
			nil, nil,
		},
		{"add route",
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
			nil,
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
			nil,
		},
		{"delete route",
			nil,
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
			nil,
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
		},
		{"keep one delete one",
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
			Routes{peeringRoute("pcx-123", "172.30.1.0/24"), peeringRoute("pcx-123", "172.30.2.0/24")},
			nil,
			Routes{peeringRoute("pcx-123", "172.30.2.0/24")},
		},
		{"keep one add one",
			Routes{peeringRoute("pcx-123", "172.30.1.0/24"), peeringRoute("pcx-123", "172.30.2.0/24")},
			Routes{peeringRoute("pcx-123", "172.30.1.0/24")},
			Routes{peeringRoute("pcx-123", "172.30.2.0/24")},
			nil,
		},
		{"keep one add one delete one",
			Routes{peeringRoute("pcx-123", "172.30.1.0/24"), peeringRoute("pcx-123", "172.30.2.0/24")},
			Routes{peeringRoute("pcx-123", "172.30.1.0/24"), peeringRoute("pcx-123", "172.30.3.0/24")},
			Routes{peeringRoute("pcx-123", "172.30.2.0/24")},
			Routes{peeringRoute("pcx-123", "172.30.3.0/24")},
		},
	}

	for _, tti := range tt {
		t.Run(tti.id, func(t *testing.T) {
			delta := ackcompare.NewDelta()
			a := createRouteTableTestResource(tti.desiredRoutes)
			b := createRouteTableTestResource(tti.latestRoutes)
			customPreCompare(delta, a, b)
			if len(tti.toAdd) == 0 && len(tti.toDelete) == 0 {
				assert.Equal(t, 0, len(delta.Differences))
			} else {
				diff := delta.Differences[0]
				diffA := diff.A.([]*svcapitypes.CreateRouteInput)
				diffB := diff.B.([]*svcapitypes.CreateRouteInput)
				assert.True(t, diff.Path.Contains("Spec.Routes"))
				assert.ElementsMatch(t, tti.desiredRoutes, diffA)
				assert.ElementsMatch(t, tti.latestRoutes, diffB)

				// Check the different routes are identified correctly
				toAdd, toDelete := filterDifferentRoutes(tti.desiredRoutes, tti.latestRoutes)
				assertRoutesIdentical(t, tti.toAdd, toAdd)
				assertRoutesIdentical(t, tti.toDelete, toDelete)
			}
		})
	}
}
