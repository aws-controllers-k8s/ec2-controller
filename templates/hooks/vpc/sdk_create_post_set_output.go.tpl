    rm.attachSecondaryCidrBlocks(ctx, desired, resp, ko)
    rm.setSpecCIDRs(ko)
    err = rm.createAttributes(ctx, &resource{ko})
    if err != nil {
        return nil, err
    } 