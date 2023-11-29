if (desired.ko.Spec.AcceptVPCPeeringRequestsFromVPCID != nil ||
    desired.ko.Spec.AcceptVPCPeeringRequestsFromVPCRefs != nil ||
    desired.ko.Spec.RejectVPCPeeringRequestsFromVPCID != nil ||
    desired.ko.Spec.RejectVPCPeeringRequestsFromVPCRefs != nil) {

	// Create an EC2 client using the AWS SDK Go v2
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Println("Error loading AWS SDK config:", err)
		return
	}

    cfg.Region = GETMESOMEFROMWHERE
	// Create an EC2 client
	ec2Client := ec2.NewFromConfig(cfg)

    // Describe all VPC Peering connections that are Pending Acceptance in the region
    input := &ec2.DescribeVpcPeeringConnectionsInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("status-code"),
				Values: []string{"pending-acceptance"},
			},
		},
	}
    resp, err := client.DescribeVpcPeeringConnections(ctx, input)
	if err != nil {
		fmt.Println("Error describing VPC peering connections:", err)
		return
	}
    peeringConnections = resp.VpcPeeringConnections

    // Iterate through each VPC Peering connection
    for _, peeringConnection := range peeringConnections {
        // Check if the peerVpcId is our VPC
        if peeringConnection.AccepterVPCInfo.VpcID == desired.ko.Spec.AcceptVPCPeeringRequestsFromVPCID {
            // Check if the VpcId is in our allow-list of VPCs
            if isVPCInWhitelist(peeringConnection.RequesterVPCInfo.VpcID) {
                // Accept the VPC Peering Request
                err := acceptVPCPeeringRequest(ec2Client, peeringConnection.VpcPeeringConnectionId)
                if err != nil {
                    return err
                }
            }
        }
        if peeringConnection.AccepterVPCInfo.VpcID == desired.ko.Spec.RejectVPCPeeringRequestsFromVPCID {
            // Check if the VpcId is in our block-list of VPCs
            if isVPCInWhitelist(peeringConnection.RequesterVPCInfo.VpcID) {
                // Reject the VPC Peering Request
                err := rejectVPCPeeringRequest(ec2Client, peeringConnection.VpcPeeringConnectionId)
                if err != nil {
                    return err
                }
            }
        }
    }
}
