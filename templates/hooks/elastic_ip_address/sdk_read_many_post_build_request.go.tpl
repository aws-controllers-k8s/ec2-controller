	if r.ko.Status.AllocationID != nil {
		input.AllocationIds = []string{*r.ko.Status.AllocationID}
	} else if r.ko.Status.PublicIP != nil {
		input.PublicIps = []string{*r.ko.Status.PublicIP}
	}