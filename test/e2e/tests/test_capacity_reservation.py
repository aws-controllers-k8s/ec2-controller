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

"""Integration tests for Capacity Reservations API.
"""

import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "capacityreservations"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

@pytest.fixture
def simple_capacity_reservation(request):
    resource_name = random_suffix_name("cr-ack-test", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["RESERVATION_NAME"] = resource_name
    replacements["INSTANCE_TYPE"] = "t2.nano"
    replacements["INSTANCE_PLATFORM"] = "Linux/UNIX"
    replacements["INSTANCE_COUNT"] = "1"
    replacements["AVAILABILITY_ZONE"] = "us-west-2a"

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']

    # Load CapacityReservation CR
    resource_data = load_ec2_resource(
        "capacity_reservation",
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
class TestCapacityReservation:
    def test_crud(self, ec2_client, simple_capacity_reservation):
        (ref, cr) = simple_capacity_reservation
        resource_id = cr["status"]["capacityReservationID"]

        # Check CapacityReservation exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_capacity_reservation(resource_id)

        # Patch the capacity reservation
        updates = {
            "spec": {"instanceCount": 2},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        capacity_reservation = ec2_validator.get_capacity_reservation(resource_id)
        assert capacity_reservation['TotalInstanceCount'] == 2
        
        resource = k8s.get_resource(ref)
        assert resource["spec"]["instanceCount"] == resource["status"]["totalInstanceCount"]

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check CapacityReservation no longer exists in AWS
        ec2_validator.assert_capacity_reservation(resource_id, exists=False)
        
    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_capacity_reservation):
        (ref, cr) = simple_capacity_reservation
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["capacityReservationID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check CapacityReservation exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_capacity_reservation(resource_id)
        
        # Check system and user tags exist for capacity reservation resource
        capacity_reservation = ec2_validator.get_capacity_reservation(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=capacity_reservation["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=capacity_reservation["Tags"],
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

        # Patch the capacity reservation, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        capacity_reservation = ec2_validator.get_capacity_reservation(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=capacity_reservation["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=capacity_reservation["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the capacity reservation resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        capacity_reservation = ec2_validator.get_capacity_reservation(resource_id)
        tags.assert_ack_system_tags(
            tags=capacity_reservation["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=capacity_reservation["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check capacity reservation no longer exists in AWS
        ec2_validator.assert_capacity_reservation(resource_id, exists=False)
