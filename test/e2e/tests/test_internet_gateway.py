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

"""Integration tests for the InternetGateway API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "internetgateways"

CREATE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@service_marker
@pytest.mark.canary
class TestInternetGateway:
    def test_create_delete(self, ec2_client):
        resource_name = random_suffix_name("ig-ack-test", 24)
        replacements = REPLACEMENT_VALUES.copy()
        replacements["INTERNET_GATEWAY_NAME"] = resource_name

        # Load Internet Gateway CR
        resource_data = load_ec2_resource(
            "internet_gateway",
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
        resource_id = resource["status"]["internetGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Internet Gateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_internet_gateway(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Internet Gateway no longer exists in AWS
        ec2_validator.assert_internet_gateway(resource_id, exists=False)

    def test_vpc_association(self, ec2_client):
        resource_name = random_suffix_name("ig-ack-test", 24)

        test_vpc = get_bootstrap_resources().EmptyVPC
        vpc_id = test_vpc.vpc_id

        replacements = REPLACEMENT_VALUES.copy()
        replacements["INTERNET_GATEWAY_NAME"] = resource_name
        replacements["VPC_ID"] = vpc_id

        # Load Internet Gateway CR
        resource_data = load_ec2_resource(
            "internet_gateway_vpc_attachment",
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
        resource_id = resource["status"]["internetGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Internet Gateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_internet_gateway(resource_id)

        # Check attachments appear on Internet Gateway
        igw = ec2_validator.get_internet_gateway(resource_id)
        assert len(igw["Attachments"]) == 1
        assert igw["Attachments"][0]["VpcId"] == vpc_id
        assert igw["Attachments"][0]["State"] == "available"

        # Patch the IGW, removing the attachment
        updates = {
            "spec": {"vpc": None},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check there are no attachments on Internet Gateway
        igw = ec2_validator.get_internet_gateway(resource_id)

        # In the case where it shows the attachment as being in detached state
        if len(igw["Attachments"]) == 1:
            assert igw["Attachments"][0]["VpcId"] == vpc_id
            assert igw["Attachments"][0]["State"] == "detached"
        else:
            # Otherwise there should be no attachment on the IGW
            assert len(igw["Attachments"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Internet Gateway no longer exists in AWS
        ec2_validator.assert_internet_gateway(resource_id, exists=False)