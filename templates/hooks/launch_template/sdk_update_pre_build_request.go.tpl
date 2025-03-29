	if delta.DifferentAt("Spec.Tags") {
		if err := syncTags(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.ID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.DefaultVersion") {
		err = rm.updateDefaultVersion(ctx, latest, nil)
		if err != nil {
			return desired, err
		}
	}

	if !delta.DifferentExcept("Spec.Tags", "Spec.DefaultVersion") {
		return desired, nil
	}
