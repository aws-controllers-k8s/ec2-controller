
	input.AddSubnetIds, input.RemoveSubnetIds = compareSubnetIDs(
		desired.ko.Spec.SubnetIDs,
		latest.ko.Spec.SubnetIDs,
	)