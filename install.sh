kubectl apply -f ../ec2-controller/config/crd/bases/ec2.services.k8s.aws_launchtemplates.yaml
kubectl apply -f ../ec2-controller/config/crd/bases/ec2.services.k8s.aws_launchtemplateversions.yaml
kubectl apply -f ../ec2-controller/sample_template/launch_template.yaml
