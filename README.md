# Test ACK service controller for Amazon Elastic Compute Cloud (EC2)

This repository contains source code for the AWS Controllers for Kubernetes
(ACK) service controller for Amazon EC2.

Please [log issues][ack-issues] and feedback on the main AWS Controllers for
Kubernetes Github project.

[ack-issues]: https://github.com/aws-controllers-k8s/community/issues

## Install the Controller

Start with the [Install an ACK Controller](https://aws-controllers-k8s.github.io/community/docs/user-docs/install/) section to install the controller into a cluster and setup necessary IAM Permissions.

*Note: it is recommended and assumed your local terminal has kubectl and AWS credentials configured to use the hosting cluster and AWS account, respectively.*

### Release Artifacts

The latest images and Helm Charts can be found in their respective ECR Public repository:
* [Images](https://gallery.ecr.aws/aws-controllers-k8s/ec2-controller)
* [Helm charts](https://gallery.ecr.aws/aws-controllers-k8s/ec2-chart)


## Create/Delete an ACK Resource

* Navigate to [test resources](https://github.com/aws-controllers-k8s/ec2-controller/tree/main/test/e2e/resources) for a list of resource `yaml` templates
* Copy the file to the local terminal and substitute `$` values. Ex: [vpc.yaml](https://github.com/aws-controllers-k8s/ec2-controller/blob/main/test/e2e/resources/vpc.yaml)

```
apiVersion: ec2.services.k8s.aws/v1alpha1
kind: VPC
metadata:
  name: $VPC_NAME
spec:
  cidrBlocks: 
  - $CIDR_BLOCK
  enableDNSSupport: $ENABLE_DNS_SUPPORT
  enableDNSHostnames: $ENABLE_DNS_HOSTNAMES
  tags:
    - key: $TAG_KEY
      value: $TAG_VALUE
```

* Create a VPC: `kubectl apply -f vpc.yaml`
* Check its status: `kubectl describe vpc/My-ACK-Resource`
* Delete the VPC: `kubectl delete -f vpc.yaml`

## Uninstall the Controller

Navigate to [Uninstall an ACK Controller](https://aws-controllers-k8s.github.io/community/docs/user-docs/cleanup/) section and substitute service values with `ec2`

## Contributing

We welcome community contributions and pull requests.

See our [contribution guide](/CONTRIBUTING.md) for more information on how to
report issues, set up a development environment, and submit code.

We adhere to the [Amazon Open Source Code of Conduct][coc].

You can also learn more about our [Governance](/GOVERNANCE.md) structure.

[coc]: https://aws.github.io/code-of-conduct

## License

This project is [licensed](/LICENSE) under the Apache-2.0 License.
