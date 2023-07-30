kubectl delete launchtemplate myfirstlaunchtemplate
kubectl patch launchtemplate myfirstlaunchtemplate -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl delete crds launchtemplates.ec2.services.k8s.aws
