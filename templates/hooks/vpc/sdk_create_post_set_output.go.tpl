    if resp.Vpc.CidrBlock != nil {
    ko.Spec.CIDRBlocks = make([]*string, 1)
    ko.Spec.CIDRBlocks[0] = resp.Vpc.CidrBlock
    }
    rm.syncCIDRBlocks(ctx, desired, &resource{ko})

    rm.setSpecCIDRs(ko)
    err = rm.createAttributes(ctx, &resource{ko})
    if err != nil {
        return nil, err
    } 
	sgDefaultRulesExist, err := rm.hasSecurityGroupDefaultRules(ctx, &resource{ko})
	if err != nil {
		return nil, err
	}
	ko.Status.SecurityGroupDefaultRulesExist = &sgDefaultRulesExist

	// If the user requested disallowing default SG rules, requeue so the
	// update path can handle the deletion. We avoid making the delete call
	// here because the runtime treats sdkCreate failures as resource
	// creation failures, which could orphan the VPC.
	if sgDefaultRulesExist && desired.ko.Spec.DisallowSecurityGroupDefaultRules != nil && *desired.ko.Spec.DisallowSecurityGroupDefaultRules {
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, aws.String("VPC created, requeue to delete default security group rules"), nil)
		err = ackrequeue.NeededAfter(fmt.Errorf("VPC created but default security group rules need to be deleted"), time.Second)
		return &resource{ko}, err
	}
