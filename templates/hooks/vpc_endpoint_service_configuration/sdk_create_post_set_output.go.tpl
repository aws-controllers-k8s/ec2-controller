
    // This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
    ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, aws.String("Reconciling to sync additional fields"), nil)
