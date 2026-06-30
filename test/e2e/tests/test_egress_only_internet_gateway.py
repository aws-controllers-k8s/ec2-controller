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

"""Integration tests for the EgressOnlyInternetGateway API.
"""

import pytest
import time
import logging

import boto3
from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "egressonlyinternetgateways"
VPC_RESOURCE_PLURAL = "vpcs"

CREATE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
WAIT_PERIOD = 30

@pytest.fixture
def simple_eigw(request, ec2_client):
    resource_name = random_suffix_name("eigw-ack-test", 24)

    test_vpc = get_bootstrap_resources().SharedTestVPC

    replacements = REPLACEMENT_VALUES.copy()
    replacements["EIGW_NAME"] = resource_name
    replacements["VPC_ID"] = test_vpc.vpc_id

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']

    # Load Egress Only Internet Gateway CR
    resource_data = load_ec2_resource(
        "egress_only_internet_gateway",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    cr = k8s.wait_resource_consumed_by_controller(ref)
    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestEgressOnlyInternetGateway:

    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud(self, ec2_client, simple_eigw):
        (ref, cr) = simple_eigw

        assert k8s.wait_on_condition(
            ref,
            "ACK.ResourceSynced",
            "True",
            wait_periods=WAIT_PERIOD,
        )

        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        eigw_id = cr["status"]["id"]

        # Check Egress Only Internet Gateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        eigw = ec2_validator.get_egress_only_internet_gateway(eigw_id)

        assert eigw is not None

        # Verify initial tags
        initial_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=eigw["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=initial_tags,
            actual=eigw["Tags"],
        )

        # Verify VPC attachment
        assert len(eigw["Attachments"]) == 1
        test_vpc = get_bootstrap_resources().SharedTestVPC
        assert eigw["Attachments"][0]["VpcId"] == test_vpc.vpc_id

        # Update tags
        updated_tags = [
            {
                "key": "updatedtagkey",
                "value": "updatedtagvalue",
            }
        ]

        k8s.patch_custom_resource(ref, {"spec": {"tags": updated_tags}})
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=WAIT_PERIOD)

        # Verify updated tags in AWS
        eigw = ec2_validator.get_egress_only_internet_gateway(eigw_id)
        expected_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=eigw["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=expected_tags,
            actual=eigw["Tags"],
        )

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Egress Only Internet Gateway no longer exists in AWS
        ec2_validator.assert_egress_only_internet_gateway(eigw_id, exists=False)

    def test_multiple_eigws_different_vpcs(self, ec2_client):
        """Regression test: creating a second EIGW on a different VPC must not
        be blocked by an existing EIGW in the same account.

        Before the fix, sdkFind called DescribeEgressOnlyInternetGateways with
        no filter when Status.ID was nil, matched the first EIGW's existing
        EIGW, and returned a spurious 'Resource already exists' terminal error.
        """
        ec2_validator = EC2Validator(ec2_client)

        # --- Create VPC A and EIGW A ---
        vpc_a_name = random_suffix_name("eigw-vpc-a", 24)
        vpc_a_replacements = REPLACEMENT_VALUES.copy()
        vpc_a_replacements["VPC_NAME"] = vpc_a_name
        vpc_a_replacements["CIDR_BLOCK"] = "10.90.0.0/16"
        vpc_a_replacements["ENABLE_DNS_SUPPORT"] = "True"
        vpc_a_replacements["ENABLE_DNS_HOSTNAMES"] = "False"
        vpc_a_replacements["ENABLE_NETWORK_ADDRESS_USAGE_METRICS"] = "False"
        vpc_a_replacements["DISALLOW_DEFAULT_SECURITY_GROUP_RULE"] = "False"
        vpc_a_replacements["TAG_KEY"] = "eigw-test"
        vpc_a_replacements["TAG_VALUE"] = "vpc-a"

        vpc_a_data = load_ec2_resource("vpc", additional_replacements=vpc_a_replacements)
        vpc_a_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, VPC_RESOURCE_PLURAL,
            vpc_a_name, namespace="default",
        )
        k8s.create_custom_resource(vpc_a_ref, vpc_a_data)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        vpc_a_cr = k8s.wait_resource_consumed_by_controller(vpc_a_ref)
        assert vpc_a_cr is not None
        assert k8s.wait_on_condition(vpc_a_ref, "ACK.ResourceSynced", "True", wait_periods=WAIT_PERIOD)
        vpc_a_cr = k8s.get_resource(vpc_a_ref)
        vpc_a_id = vpc_a_cr["status"]["vpcID"]

        eigw_a_name = random_suffix_name("eigw-a", 24)
        eigw_a_replacements = REPLACEMENT_VALUES.copy()
        eigw_a_replacements["EIGW_NAME"] = eigw_a_name
        eigw_a_replacements["VPC_ID"] = vpc_a_id
        eigw_a_replacements["TAG_KEY"] = "eigw-test"
        eigw_a_replacements["TAG_VALUE"] = "eigw-a"

        eigw_a_data = load_ec2_resource(
            "egress_only_internet_gateway",
            additional_replacements=eigw_a_replacements,
        )
        eigw_a_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            eigw_a_name, namespace="default",
        )
        k8s.create_custom_resource(eigw_a_ref, eigw_a_data)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(eigw_a_ref, "ACK.ResourceSynced", "True", wait_periods=WAIT_PERIOD)
        eigw_a_cr = k8s.get_resource(eigw_a_ref)
        eigw_a_id = eigw_a_cr["status"]["id"]
        assert eigw_a_id is not None

        # --- Create VPC B and EIGW B (must not collide with EIGW A) ---
        vpc_b_name = random_suffix_name("eigw-vpc-b", 24)
        vpc_b_replacements = REPLACEMENT_VALUES.copy()
        vpc_b_replacements["VPC_NAME"] = vpc_b_name
        vpc_b_replacements["CIDR_BLOCK"] = "10.91.0.0/16"
        vpc_b_replacements["ENABLE_DNS_SUPPORT"] = "True"
        vpc_b_replacements["ENABLE_DNS_HOSTNAMES"] = "False"
        vpc_b_replacements["ENABLE_NETWORK_ADDRESS_USAGE_METRICS"] = "False"
        vpc_b_replacements["DISALLOW_DEFAULT_SECURITY_GROUP_RULE"] = "False"
        vpc_b_replacements["TAG_KEY"] = "eigw-test"
        vpc_b_replacements["TAG_VALUE"] = "vpc-b"

        vpc_b_data = load_ec2_resource("vpc", additional_replacements=vpc_b_replacements)
        vpc_b_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, VPC_RESOURCE_PLURAL,
            vpc_b_name, namespace="default",
        )
        k8s.create_custom_resource(vpc_b_ref, vpc_b_data)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        vpc_b_cr = k8s.wait_resource_consumed_by_controller(vpc_b_ref)
        assert vpc_b_cr is not None
        assert k8s.wait_on_condition(vpc_b_ref, "ACK.ResourceSynced", "True", wait_periods=WAIT_PERIOD)
        vpc_b_cr = k8s.get_resource(vpc_b_ref)
        vpc_b_id = vpc_b_cr["status"]["vpcID"]

        eigw_b_name = random_suffix_name("eigw-b", 24)
        eigw_b_replacements = REPLACEMENT_VALUES.copy()
        eigw_b_replacements["EIGW_NAME"] = eigw_b_name
        eigw_b_replacements["VPC_ID"] = vpc_b_id
        eigw_b_replacements["TAG_KEY"] = "eigw-test"
        eigw_b_replacements["TAG_VALUE"] = "eigw-b"

        eigw_b_data = load_ec2_resource(
            "egress_only_internet_gateway",
            additional_replacements=eigw_b_replacements,
        )
        eigw_b_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            eigw_b_name, namespace="default",
        )
        k8s.create_custom_resource(eigw_b_ref, eigw_b_data)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # This is the key assertion: EIGW B must sync successfully,
        # NOT hit "Resource already exists" terminal error.
        assert k8s.wait_on_condition(
            eigw_b_ref,
            "ACK.ResourceSynced",
            "True",
            wait_periods=WAIT_PERIOD,
        ), "EIGW B should sync successfully; before the fix it would hit 'Resource already exists'"

        eigw_b_cr = k8s.get_resource(eigw_b_ref)
        eigw_b_id = eigw_b_cr["status"]["id"]
        assert eigw_b_id is not None

        # Verify both EIGWs exist in AWS and are on different VPCs
        eigw_a_aws = ec2_validator.get_egress_only_internet_gateway(eigw_a_id)
        eigw_b_aws = ec2_validator.get_egress_only_internet_gateway(eigw_b_id)
        assert eigw_a_aws is not None
        assert eigw_b_aws is not None
        assert eigw_a_id != eigw_b_id, "Each VPC must get its own EIGW"
        assert eigw_a_aws["Attachments"][0]["VpcId"] == vpc_a_id
        assert eigw_b_aws["Attachments"][0]["VpcId"] == vpc_b_id

        # --- Cleanup (reverse order) ---
        k8s.delete_custom_resource(eigw_b_ref, 3, 10)
        k8s.delete_custom_resource(eigw_a_ref, 3, 10)
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        k8s.delete_custom_resource(vpc_b_ref, 3, 10)
        k8s.delete_custom_resource(vpc_a_ref, 3, 10)
        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        ec2_validator.assert_egress_only_internet_gateway(eigw_a_id, exists=False)
        ec2_validator.assert_egress_only_internet_gateway(eigw_b_id, exists=False)
