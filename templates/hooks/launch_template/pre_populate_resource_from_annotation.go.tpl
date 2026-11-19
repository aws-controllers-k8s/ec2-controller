	if id, ok := fields["id"]; ok {
		r.ko.Status.ID = &id
		if name, ok := fields["name"]; ok {
			r.ko.Spec.Name = &name
		}
		return nil
	}
