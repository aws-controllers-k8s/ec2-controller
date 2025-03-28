
	err = rm.findLaunchTemplateVersion(ctx, r, ko)
	if err != nil {
		return &resource{ko}, nil
	}
