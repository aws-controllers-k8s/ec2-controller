    if rm.requiredFieldsMissingForCreateRoute(&resource{ko}) {
        return nil, ackerr.NotFound
    }
    if err := rm.createRoutes(ctx, &resource{ko}); err != nil {
        return nil, err
    }