	if len(a.ko.Spec.IngressRules) != len(b.ko.Spec.IngressRules) {
		delta.Add("Spec.IngressRules", a.ko.Spec.IngressRules, b.ko.Spec.IngressRules)
	} else {
		for _, aRule := range a.ko.Spec.IngressRules {
			if !containsRule(b.ko.Spec.IngressRules, aRule) {
				delta.Add("Spec.IngressRules", a.ko.Spec.IngressRules, b.ko.Spec.IngressRules)
				break
			}
		}
	}
	if len(a.ko.Spec.EgressRules) != len(b.ko.Spec.EgressRules) {
		delta.Add("Spec.EgressRules", a.ko.Spec.EgressRules, b.ko.Spec.EgressRules)
	} else {
		for _, aRule := range a.ko.Spec.EgressRules {
			if !containsRule(b.ko.Spec.EgressRules, aRule) {
				delta.Add("Spec.EgressRules", a.ko.Spec.EgressRules, b.ko.Spec.EgressRules)
				break
			}
		}
	}
