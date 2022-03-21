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

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "vpcs"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

def get_vpc_attribute(ec2_client, vpc_id: str, attribute_name: str) -> dict:
    return ec2_client.describe_vpc_attribute(Attribute=attribute_name, VpcId=vpc_id)
    
def get_dns_support(ec2_client, vpc_id: str) -> bool:
    attribute = get_vpc_attribute(ec2_client, vpc_id, 'enableDnsSupport')
    return attribute['EnableDnsSupport']['Value']

def get_dns_hostnames(ec2_client, vpc_id: str) -> bool:
    attribute = get_vpc_attribute(ec2_client, vpc_id, 'enableDnsHostnames')
    return attribute['EnableDnsHostnames']['Value']

@service_marker
@pytest.mark.canary
class TestVpc:
    def test_create_delete(self, ec2_client):
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

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

    def test_enable_attributes(self, ec2_client):
        resource_name = random_suffix_name("vpc-ack-test", 24)
        replacements = REPLACEMENT_VALUES.copy()
        replacements["VPC_NAME"] = resource_name
        replacements["CIDR_BLOCK"] = "10.0.0.0/16"
        replacements["ENABLE_DNS_SUPPORT"] = "True"
        replacements["ENABLE_DNS_HOSTNAMES"] = "True"

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

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Assert the attributes are set correctly
        assert get_dns_support(ec2_client, resource_id)
        assert get_dns_hostnames(ec2_client, resource_id)

        # Disable the DNS support
        updates = {
            "spec": {"enableDNSSupport": False}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Assert DNS support has been updated
        assert not get_dns_support(ec2_client, resource_id)
        assert get_dns_hostnames(ec2_client, resource_id)

        # Disable the DNS hostname
        updates = {
            "spec": {"enableDNSHostnames": False}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Assert DNS hostname has been updated
        assert not get_dns_support(ec2_client, resource_id)
        assert not get_dns_hostnames(ec2_client, resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)
