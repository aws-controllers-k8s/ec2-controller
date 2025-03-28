	err = rm.setLatestLaunchTemplateAttributes(ctx, r, ko)
	if err != nil {
		return &resource{ko}, nil
	}
