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
	if delta.DifferentAt("Spec.DefaultVersion") {
		defer func() {
			err = rm.updateDefaultVersion(ctx, desired)
		}()
	}

	if !delta.DifferentExcept("Spec.Tags", "Spec.DefaultVersion") {
		return desired, nil
	}
