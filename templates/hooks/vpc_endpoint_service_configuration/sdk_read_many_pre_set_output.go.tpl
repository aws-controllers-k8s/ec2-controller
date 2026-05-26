
	// Filter out regions that are being removed (Deleting/Deleted/Failed/Closed)
	// so the delta comparison doesn't try to re-remove them. Pending regions are
	// kept so newly-added regions aren't re-added while AWS is still activating them.
	for i := range resp.ServiceConfigurations {
		activeRegions := make([]svcsdktypes.SupportedRegionDetail, 0, len(resp.ServiceConfigurations[i].SupportedRegions))
		for _, r := range resp.ServiceConfigurations[i].SupportedRegions {
			if r.ServiceState == nil {
				continue
			}
			switch *r.ServiceState {
			case "Deleting", "Deleted", "Failed", "Closed":
				continue
			default:
				activeRegions = append(activeRegions, r)
			}
		}
		resp.ServiceConfigurations[i].SupportedRegions = activeRegions
	}
