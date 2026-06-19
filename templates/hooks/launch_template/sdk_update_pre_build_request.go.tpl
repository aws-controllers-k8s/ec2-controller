	if delta.DifferentAt("Spec.Tags") {
		if err := syncTags(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.ID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}
	// We want to update the defaultVersion after we create the new
	// version if needed.
	// Wondering how this works? find out in https://go.dev/play/p/10QSDg2xbTB
	//
	// Only call ModifyLaunchTemplate when the user actually set a desired
	// DefaultVersion. An adopted template reports its server-side default
	// (e.g. 1) while the desired spec leaves it unset, producing a spurious
	// delta; without this guard updateDefaultVersion would fail with
	// "field DefaultVersion is required" before late initialization can
	// populate it.
	if delta.DifferentAt("Spec.DefaultVersion") && desired.ko.Spec.DefaultVersion != nil {
		defer func() {
			err = rm.updateDefaultVersion(ctx, desired)
		}()
	}

	if !delta.DifferentExcept("Spec.Tags", "Spec.DefaultVersion") {
		return desired, nil
	}
