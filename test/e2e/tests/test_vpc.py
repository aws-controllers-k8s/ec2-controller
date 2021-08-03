# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the Vpc API.
"""

import boto3
import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_PLURAL = "vpcs"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def ec2_client():
    return boto3.client("ec2")

@service_marker
@pytest.mark.canary
class TestVpc:
    def get_vpc(self, ec2_client, vpc_id: str) -> dict:
        try:
            resp = ec2_client.describe_vpcs()
        except Exception as e:
            logging.debug(e)
            return None

        vpcs = resp["Vpcs"]
        for vpc in vpcs:
            if vpc["VpcId"] == vpc_id:
                return vpc

        return None

    def vpc_exists(self, ec2_client, vpc_id: str) -> bool:
        return self.get_vpc(ec2_client, vpc_id) is not None

    def test_smoke(self, ec2_client):
        resource_name = random_suffix_name("vpc-ack-test", 24)
        replacements = REPLACEMENT_VALUES.copy()
        replacements["VPC_NAME"] = resource_name
        replacements["CIDR_BLOCK"] = "10.0.0.0/16"

        # Load VPC CR
        resource_data = load_ec2_resource(
            "vpc",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["vpcID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC exists
        exists = self.vpc_exists(ec2_client, resource_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC doesn't exist
        exists = self.vpc_exists(ec2_client, resource_id)
        assert not exists
