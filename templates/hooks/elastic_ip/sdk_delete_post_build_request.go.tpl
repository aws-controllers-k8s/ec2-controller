    // PublicIP and AllocationID are two ways of identifying the same resource
    // depending on whether they are included as part of EC2-Classic or EC2-VPC,
    // respectively. As EC2-VPC is the preferred method, we should attempt to
    // use the AllocationID field whenever possible.
    if input.PublicIp != nil && input.AllocationId != nil {
		input.PublicIp = nil
	}