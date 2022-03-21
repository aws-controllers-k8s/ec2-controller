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

"""Integration tests for the SecurityGroup API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "securitygroups"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@service_marker
@pytest.mark.canary
class TestSecurityGroup:
    def test_create_delete(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("security-group-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id

        test_resource_values["SECURITY_GROUP_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        test_resource_values["SECURITY_GROUP_DESCRIPTION"] = "TestSecurityGroup-create-delete"

        # Load Security Group CR
        resource_data = load_ec2_resource(
            "security_group",
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
        resource_id = resource["status"]["id"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Security Group exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Security Group no longer exists in AWS
        ec2_validator.assert_security_group(resource_id, exists=False)

    def test_terminal_condition(self):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("security-group-fail", 24)

        test_resource_values["SECURITY_GROUP_NAME"] = resource_name
        test_resource_values["VPC_ID"] = "InvalidVpcId"
        test_resource_values["SECURITY_GROUP_DESCRIPTION"] = "TestSecurityGroup-terminal"

        # Load Security Group CR
        resource_data = load_ec2_resource(
            "security_group",
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

        expected_msg = "InvalidVpcID.NotFound: The vpc ID 'InvalidVpcId' does not exist"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        #   InvalidVpcID.NotFound: The vpc ID 'InvalidVpcId' does not exist
        #   status code: 400, request id: 5801fc80-67cf-465f-8b83-5e02d517d554
        # This check only verifies the error message; the request hash is irrelevant and therefore can be ignored.
        assert expected_msg in terminal_condition['message']