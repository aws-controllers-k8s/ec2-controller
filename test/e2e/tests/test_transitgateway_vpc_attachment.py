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

"""Integration tests for the Vpc Endpoint API.
"""

from os import environ
import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "transitgatewayvpcattachments"

CREATE_WAIT_AFTER_SECONDS = 30
MODIFY_WAIT_AFTER_SECONDS = 30
WAIT_PERIOD = 30

@pytest.fixture
def simple_tgw_attachment(request, ec2_client):
    resource_name = random_suffix_name("tgw-attachment-test", 24)
    
    test_vpc = get_bootstrap_resources().SharedTestVPC
    test_tgw = get_bootstrap_resources().TestTransitGateway

    tgw_id = test_tgw.transit_gateway_id

    ec2_validator = EC2Validator(ec2_client)
    is_available = ec2_validator.wait_transit_gateway_state(tgw_id=tgw_id, state='available')
    assert is_available
    
    replacements = REPLACEMENT_VALUES.copy()

    replacements["TGWVA_NAME"] = resource_name
    replacements["VPC_ID"] = test_vpc.vpc_id
    replacements["TGW_ID"] = test_tgw.transit_gateway_id
    replacements["SUBNET_ID"] = test_vpc.public_subnets.subnet_ids[0]

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']

    # Load TGW Attachment CR
    resource_data = load_ec2_resource(
        "transitgateway_vpc_attachment",
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
class TestTransitGatewayVPCAttachment:

    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud(self, ec2_client, simple_tgw_attachment):
        (ref, cr) = simple_tgw_attachment

        assert k8s.wait_on_condition(
            ref,
            "ACK.ResourceSynced",
            "True",
            wait_periods=WAIT_PERIOD,
        )
        
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        attachment_id = cr["status"]["id"]

        # Check TGW Attachment exists and verify initial tags
        ec2_validator = EC2Validator(ec2_client)
        attachment = ec2_validator.get_transit_gateway_vpc_attachment(attachment_id)
        
        assert attachment is not None
        
        initial_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=attachment["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=initial_tags,
            actual=attachment["Tags"],
        )

        # Update tags
        updated_tags = [
            {
                "key": "updatedtagkey",
                "value": "updatedtagvalue",
            }
        ]

        k8s.patch_custom_resource(ref, {"spec": {"tags": updated_tags}})
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Verify updated tags
        attachment = ec2_validator.get_transit_gateway_vpc_attachment(attachment_id)
        expected_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=attachment["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=expected_tags,
            actual=attachment["Tags"],
        )

        # Update options
        # dns support is enabled by default
        updates = {
            "spec": {
                "options": {
                    "dnsSupport": "disable",
                }
            }
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=WAIT_PERIOD)

        # Verify the update in AWS
        ec2_validator = EC2Validator(ec2_client)
        attachment = ec2_validator.get_transit_gateway_vpc_attachment(attachment_id)
        
        assert attachment["Options"]["DnsSupport"] == "disable"

        # Update subnet ids
        test_vpc = get_bootstrap_resources().SharedTestVPC
        updates = {
            "spec": {
                "subnetIDs": test_vpc.public_subnets.subnet_ids
            }
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=WAIT_PERIOD)

        # Verify the update in AWS
        ec2_validator = EC2Validator(ec2_client)
        attachment = ec2_validator.get_transit_gateway_vpc_attachment(attachment_id)
        
        assert set(attachment["SubnetIds"]) == set(test_vpc.public_subnets.subnet_ids)
