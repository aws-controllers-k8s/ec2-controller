
	mirrorSecurityGroupIDs(ko)

	if vpcEndpointPending(&resource{ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse,
			aws.String("VPC endpoint is "+*ko.Status.State+", waiting for it to become available"), nil)
	}
