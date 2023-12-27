
	if r.ko.Spec.AllowedPrincipals != nil {
		for i := range r.ko.Spec.AllowedPrincipals {
			f0 := r.ko.Spec.AllowedPrincipals[i]
			ko.Spec.AllowedPrincipals[i] = f0
		}
	} else {
		ko.Spec.AllowedPrincipals = nil
	}
	rlog.Debug("AAAAAAAAAAAAAAAAAAA sdkFind", "ko.Spec.AllowedPrincipals", ko.Spec.AllowedPrincipals)	