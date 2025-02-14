# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.
"""Bootstraps the resources required to run EC2 integration tests.
"""

import logging

from acktest.bootstrapping import Resources, BootstrapFailureException
from acktest.bootstrapping.elbv2 import NetworkLoadBalancer
from acktest.bootstrapping.vpc import VPC
from acktest.bootstrapping.s3 import Bucket
from e2e import bootstrap_directory
from e2e.bootstrap_resources import BootstrapResources

def service_bootstrap() -> Resources:
    logging.getLogger().setLevel(logging.INFO)

    resources = BootstrapResources(
        SharedTestVPC=VPC(
            name_prefix="e2e-test-vpc", 
            num_public_subnet=2,
            num_private_subnet=0
        ),
        FlowLogsBucket=Bucket(
            "ack-ec2-controller-flow-log-tests",
        ),
        NetworkLoadBalancer=NetworkLoadBalancer("e2e-vpc-ep-service-test"),
        AdoptedVPC=VPC(name_prefix="e2e-adopted-vpc", num_public_subnet=1, num_private_subnet=0)
    )

    try:
        resources.bootstrap()
    except BootstrapFailureException:
        exit(254)

    return resources

if __name__ == "__main__":
    config = service_bootstrap()
    # Write config to current directory by default
    config.serialize(bootstrap_directory)
