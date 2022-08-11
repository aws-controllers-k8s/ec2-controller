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

    def test_rules_create_update_delete(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("sec-group-rules", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id

        test_resource_values["SECURITY_GROUP_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        test_resource_values["SECURITY_GROUP_DESCRIPTION"] = "TestSecurityGroupRule-create-delete"
        
        # Create Security Group CR with ingress rule
        test_resource_values["IP_PROTOCOL"] = "tcp"
        test_resource_values["FROM_PORT"] = "80"
        test_resource_values["TO_PORT"] = "80"
        test_resource_values["CIDR_IP"] = "172.31.0.0/16"
        test_resource_values["DESCRIPTION_INGRESS"] = "test ingress rule"

        # Load Security Group CR
        resource_data = load_ec2_resource(
            "security_group_rule",
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

        # Check resource is late initialized successfully (sets default egress rule)
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check Security Group exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id)

        # Check ingress rule added and default egress rule present
        # default egress rule will be present iff user has NOT specified their own egress rules
        assert len(resource["status"]["rules"]) == 2
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 1
        assert len(sg_group["IpPermissionsEgress"]) == 1

        # Check default egress rule data
        assert sg_group["IpPermissionsEgress"][0]["IpProtocol"] == "-1"
        assert sg_group["IpPermissionsEgress"][0]["IpRanges"][0]["CidrIp"] == "0.0.0.0/0"

        # Add Egress rule via patch
        new_egress_rule = {
                        "ipProtocol": "tcp",
                        "fromPort": 25,
                        "toPort": 25,
                        "ipRanges": [
                            {
                                "cidrIP": "172.31.0.0/16",
                                "description": "test egress update"
                            }
                        ]
        }
        patch = {"spec": {"egressRules":[new_egress_rule]}}
        _ = k8s.patch_custom_resource(ref, patch)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check resource gets into synced state
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check ingress and egress rules exist
        assert len(resource["status"]["rules"]) == 2
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 1
        assert len(sg_group["IpPermissionsEgress"]) == 1
        
        # Check egress rule data (i.e. ensure default egress rule removed)
        assert sg_group["IpPermissionsEgress"][0]["IpProtocol"] == "tcp"
        assert sg_group["IpPermissionsEgress"][0]["FromPort"] == 25
        assert sg_group["IpPermissionsEgress"][0]["ToPort"] == 25
        assert sg_group["IpPermissionsEgress"][0]["IpRanges"][0]["Description"] == "test egress update"

        # Remove Ingress rule
        patch = {"spec": {"ingressRules":[]}}
        _ = k8s.patch_custom_resource(ref, patch)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # assert patched state
        resource = k8s.get_resource(ref)
        assert len(resource['status']['rules']) == 1

        # Check ingress rule removed; egress rule remains
        assert len(resource["status"]["rules"]) == 1
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 0
        assert len(sg_group["IpPermissionsEgress"]) == 1

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Security Group no longer exists in AWS
        # Deleting Security Group will also delete rules
        ec2_validator.assert_security_group(resource_id, exists=False)