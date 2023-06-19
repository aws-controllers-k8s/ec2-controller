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