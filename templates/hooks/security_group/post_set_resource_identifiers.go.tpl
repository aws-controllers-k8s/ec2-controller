	identifierName, identifierNameOk := identifier.AdditionalKeys["name"]
	if identifierNameOk {
		r.ko.Spec.Name = &identifierName
	}
