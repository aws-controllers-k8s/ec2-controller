
	if delta.DifferentAt("Spec.Tags") {
		if err := syncTags(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.ID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	// Only continue if something other than Tags has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags") {
		return desired, nil
	}

	if *latest.ko.Status.State != string(svcsdktypes.TransitGatewayAttachmentStateAvailable) {
		
	}
