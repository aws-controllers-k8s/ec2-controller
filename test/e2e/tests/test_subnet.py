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

"""Integration tests for the Subnet API.
"""

import boto3
import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_PLURAL = "subnets"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

test_resource_values = REPLACEMENT_VALUES.copy()


@pytest.fixture(scope="module")
def ec2_client():
    return boto3.client("ec2")

@pytest.fixture(scope="module")
def vpc_resource():
    resource_name = random_suffix_name("vpc-for-subnet", 24)
    test_resource_values["VPC_NAME"] = resource_name
    test_resource_values["CIDR_BLOCK"] = "10.0.0.0/16"

    resource_data = load_ec2_resource(
        "vpc",
        additional_replacements=test_resource_values,
    )
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, "vpcs",
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    resource = k8s.get_resource(ref)
    test_resource_values["VPC_ID"] = resource["status"]["vpcID"]

    yield ref, cr

    k8s.delete_custom_resource(ref)

@service_marker
@pytest.mark.canary
class TestSubnet:
    def get_subnet(self, ec2_client, subnet_id: str) -> dict:
        try:
            resp = ec2_client.describe_subnets()
        except Exception as e:
            logging.debug(e)
            return None

        subnets = resp["Subnets"]
        for subnet in subnets:
            if subnet["SubnetId"] == subnet_id:
                return subnet

        return None

    def subnet_exists(self, ec2_client, subnet_id: str) -> bool:
        return self.get_subnet(ec2_client, subnet_id) is not None

    def test_crud(self, ec2_client, vpc_resource):
        resource_name = random_suffix_name("subnet-crud", 24)
        test_resource_values["SUBNET_NAME"] = resource_name

        # Load Subnet CR
        resource_data = load_ec2_resource(
            "subnet",
            additional_replacements=test_resource_values,
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
        resource_id = resource["status"]["subnetID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Subnet exists
        exists = self.subnet_exists(ec2_client, resource_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC doesn't exist
        exists = self.subnet_exists(ec2_client, resource_id)
        assert not exists

    def test_terminal_condition(self, vpc_resource):
        resource_name = random_suffix_name("subnet-fail", 24)
        test_resource_values["SUBNET_NAME"] = resource_name
        test_resource_values["VPC_ID"] = "InvalidVpcId"

        # Load Subnet CR
        resource_data = load_ec2_resource(
            "subnet",
            additional_replacements=test_resource_values,
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

        assert k8s.assert_condition_state_message(
            ref, "ACK.Terminal", "True", "The vpc ID 'InvalidVpcId' does not exist"
        )
