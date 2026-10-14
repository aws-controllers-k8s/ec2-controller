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

"""Integration tests for the Transit Gateway API.
"""

import boto3
import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "transitgateways"

DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

## A TGW cannot be deleted or modified while it is still "pending", and
## provisioning routinely takes several minutes. The read path requeues with
## "resource is pending" until the TGW reports "available", so
## ACK.ResourceSynced=True is exactly that precondition -- wait on it rather than
## sleeping a fixed interval that may or may not be long enough.
SYNCED_WAIT_PERIODS = 8

@pytest.fixture
def simple_transit_gateway(request):
    resource_name = random_suffix_name("tgw-ack-test", 24)
    replacements = REPLACEMENT_VALUES.copy()
    replacements["TGW_NAME"] = resource_name

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']

    # Load TGW CR
    resource_data = load_ec2_resource(
        "transitgateway",
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

    # Every test using this fixture goes on to delete or modify the TGW, neither
    # of which AWS permits while it is "pending". Gate on the controller
    # reporting it synced, then re-read so the yielded CR carries the settled
    # status.
    assert k8s.wait_on_condition(
        ref, "ACK.ResourceSynced", "True", wait_periods=SYNCED_WAIT_PERIODS,
    )
    cr = k8s.get_resource(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestTGW:
    def test_create_delete(self, ec2_client, simple_transit_gateway):
        (ref, cr) = simple_transit_gateway
        resource_id = cr["status"]["transitGatewayID"]

        # Check TGW exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_transit_gateway(resource_id)

        # Delete k8s resource. The fixture has already waited for the TGW to
        # leave "pending", so the delete proceeds on the first reconcile.
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check TGW no longer exists in AWS
        ec2_validator.assert_transit_gateway(resource_id, exists=False)

    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_transit_gateway):
        (ref, cr) = simple_transit_gateway
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["transitGatewayID"]

        # Check TransitGateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_transit_gateway(resource_id)
        
        # Check system and user tags exist for transit gateway resource
        transit_gateway = ec2_validator.get_transit_gateway(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=transit_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=transit_gateway["Tags"],
        )
        
        # Only user tags should be present in Spec
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "initialtagkey"
        assert resource["spec"]["tags"][0]["value"] == "initialtagvalue"

        # Update tags
        update_tags = [
                {
                    "key": "updatedtagkey",
                    "value": "updatedtagvalue",
                }
            ]

        # Patch the TransitGateway, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        transit_gateway = ec2_validator.get_transit_gateway(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=transit_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=transit_gateway["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the TransitGateway resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        transit_gateway = ec2_validator.get_transit_gateway(resource_id)
        tags.assert_ack_system_tags(
            tags=transit_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=transit_gateway["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check TransitGateway no longer exists in AWS
        ec2_validator.assert_transit_gateway(resource_id, exists=False)
