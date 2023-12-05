
    if r.ko.Status.Status != nil &&
        r.ko.Status.Status.Code != nil &&
        *r.ko.Status.Status.Code == "pending-acceptance" {
        r.ko.Spec.AcceptRequest = aws.Bool(true)
    } else {
        r.ko.Spec.AcceptRequest = aws.Bool(false)
    }
