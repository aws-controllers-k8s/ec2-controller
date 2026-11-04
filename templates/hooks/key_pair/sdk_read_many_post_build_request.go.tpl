	// DescribeKeyPairs rejects KeyNames and KeyPairIds in the same call. The
	// generated payload reads by ID once Status.KeyPairID is set; read by name
	// only before that.
	if r.ko.Status.KeyPairID == nil {
		input.KeyNames = []string{*r.ko.Spec.KeyName}
	}
