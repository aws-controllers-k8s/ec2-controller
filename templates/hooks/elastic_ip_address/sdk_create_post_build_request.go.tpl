    updateTagSpecificationsInCreateRequest(desired, input)
    // EC2-VPC only supports setting Domain to "vpc"
    input.Domain = svcsdktypes.DomainTypeVpc