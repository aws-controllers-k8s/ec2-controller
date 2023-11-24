if desired.ko.Spec.AcceptVPCPeeringRequestsFromVPCID != nil || desired.ko.Spec.AcceptVPCPeeringRequestsFromVPCRefs != nil {
    // Use the AWS SDK Go v2 to describe all VPC Peering connections in the region
    peeringConnections, err := describeAllVPCPeeringConnections()
    if err != nil {
        // Handle error, e.g., log or return
        return err
    }

    // Create an EC2 client using the AWS SDK Go v2
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        // Handle error, e.g., log or return
        return err
    }

    ec2Client := ec2.NewFromConfig(cfg)

    // Iterate through each VPC Peering connection
    for _, peeringConnection := range peeringConnections {
        // Check if there are any Pending Acceptance VPC Peering Requests
        if peeringConnection.Status.Code == "pending-acceptance" {
            // Check if the peerVpcId is our VPC
            if peeringConnection.AccepterVPCInfo.VpcID == desired.ko.Spec.AcceptVPCPeeringRequestsFromVPCID {
                // Check if the VpcId is in our whitelist of VPCs
                if isVPCInWhitelist(peeringConnection.RequesterVPCInfo.VpcID) {
                    // Accept the VPC Peering Request with AWS SDK Go
                    err := acceptVPCPeeringRequest(ec2Client, peeringConnection.ID)
                    if err != nil {
                        // Handle error, e.g., log or return
                        return err
                    }
                }
            }
        }
    }
}
