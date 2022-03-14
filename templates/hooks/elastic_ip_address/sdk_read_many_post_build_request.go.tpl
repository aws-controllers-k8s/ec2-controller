	if r.ko.Status.AllocationID != nil {
		input.SetAllocationIds([]*string{r.ko.Status.AllocationID})
	} else if r.ko.Status.PublicIP != nil {
		input.SetPublicIps([]*string{r.ko.Status.PublicIP})
	}