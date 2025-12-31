
    // This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
    return &resource{ko}, ackrequeue.Needed(errors.New("reconcile update only fields."))