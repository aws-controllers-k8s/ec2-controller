
    // set the instance count in spec from the total instance count in status,
	// without this there's no diff detected for this field in the desired object and latest state in aws 
	// causing update calls to have no effect at all
	if ko.Status.TotalInstanceCount != nil {
		ko.Spec.InstanceCount = ko.Status.TotalInstanceCount
	}

	// the AdditionalInfo field is not returned by DescribeCapacityReservations API
	// so we must explicitly set it to nil before returning ko
	ko.Spec.AdditionalInfo = nil
