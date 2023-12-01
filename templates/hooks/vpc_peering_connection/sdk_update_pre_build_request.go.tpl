  
  if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

  // Only continue if something other than Tags has changed in the Spec
  if !delta.DifferentExcept("Spec.Tags") {
      return desired, nil
  }