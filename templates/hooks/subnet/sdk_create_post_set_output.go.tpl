    if err = rm.createRouteTableAssociations(ctx, &resource{ko}); err != nil {
        return nil, err
    }