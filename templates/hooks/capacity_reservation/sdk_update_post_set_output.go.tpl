
	// Explicitly call sdkFind to fetch the latest resource state
	latestCopy, err := rm.sdkFind(ctx, desired)
	if err != nil {
		return nil, err
	}

	ko.Status.AvailableInstanceCount = latestCopy.ko.Status.AvailableInstanceCount
	ko.Status.TotalInstanceCount = latestCopy.ko.Status.TotalInstanceCount
	ko.Status.State = latestCopy.ko.Status.State
