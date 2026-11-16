	updateTagSpecificationsInCreateRequest(desired, input)
	if desired.ko.Spec.PublicKeyMaterial != nil {
		input.PublicKeyMaterial = []byte(*desired.ko.Spec.PublicKeyMaterial)
	}
