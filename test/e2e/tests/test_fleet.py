# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#     http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for Managed Prefix List API.
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
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "fleets"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

FLEET_TAG_KEY = "owner"
FLEET_TAG_VAL = "ack-controller"
    
def get_ami_id(ec2_client):
    try:
        # Use latest AL2
        resp = ec2_client.describe_images(
            Owners=['amazon'],
            Filters=[
                {"Name": "architecture", "Values": ['x86_64']},
                {"Name": "state", "Values": ['available']},
                {"Name": "virtualization-type", "Values": ['hvm']},
                ],
        )
        for image in resp['Images']:
            if 'Description' in image:
                if "Amazon Linux 2 Kernel" in image['Description']:
                    return image['ImageId']
    except Exception as e:
        logging.debug(e)


@pytest.fixture(scope="module")
def ec2_validator():
    """Fixture to provide EC2 validator for AWS API calls."""
    ec2_client = boto3.client("ec2")
    return EC2Validator(ec2_client)

@pytest.fixture
def standard_launch_template(ec2_client):
    resource_name = random_suffix_name("lt-ack-test", 24)
    resource_file = "launch_template"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["LAUNCH_TEMPLATE_NAME"] = resource_name

    # Load LaunchTemplate CR
    resource_data = load_ec2_resource(
        resource_file,
        additional_replacements=replacements,
    )
    ami_id = get_ami_id(ec2_client)
    resource_data["spec"]["data"]["imageID"] = ami_id
    resource_data["spec"]["data"]["instanceType"] = 't3.nano'

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, "launchtemplates",
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    cr = k8s.wait_resource_consumed_by_controller(ref)
    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Teardown
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@pytest.fixture
def simple_fleet(standard_launch_template, request):
    resource_name = random_suffix_name("fleet", 32)     
    
    (_, launch_template) = standard_launch_template
    
    launch_template_id = launch_template["status"]["id"]

    test_resource_values = REPLACEMENT_VALUES.copy()
    test_resource_values["FLEET_NAME"] = resource_name
    test_resource_values["TOTAL_TARGET_CAPACITY"]  = "1"
    test_resource_values["DEFAULT_TARGET_CAPACITY_TYPE"] = "spot"
    test_resource_values["LAUNCH_TEMPLATE_ID"] = launch_template_id
    test_resource_values["LAUNCH_TEMPLATE_VERSION"] = "'1'"
    test_resource_values["FLEET_TAG_KEY"] = FLEET_TAG_KEY
    test_resource_values["FLEET_TAG_VAL"] = FLEET_TAG_VAL

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'fleet_type' in data:
            test_resource_values["FLEET_TYPE"] = data['fleet_type']

    # Load the resource
    resource_data = load_ec2_resource(
        "fleet",
        additional_replacements=test_resource_values,
    )

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

    # Teardown
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestFleets:

    @pytest.mark.resource_data({'fleet_type': 'instant'})
    def test_crud_instant(self, simple_fleet, ec2_validator):
        """
        Test creation, update and deletion of an Fleet of type instant.
        Fleets of type instant do not support fleet updates, so this test verifies tag updates, and verifies that other update types are blocked and result in terminal conditions.
        """
        (ref, cr) = simple_fleet

        resource = k8s.get_resource(ref)
        fleet_id = cr["status"]["fleetID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Fleet exists
        fleet = ec2_validator.get_fleet(fleet_id)
        assert fleet is not None
        
        # Check system and user tags exist for fleet resource
        user_tags = {
            FLEET_TAG_KEY: FLEET_TAG_VAL
        }
        tags.assert_ack_system_tags(
            tags=fleet["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=fleet["Tags"],
        )
        
        # Only user tags should be present in Spec
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == FLEET_TAG_KEY
        assert resource["spec"]["tags"][0]["value"] == FLEET_TAG_VAL

        # Update tags
        update_tags = [
                {
                    "key": "updatedtagkey",
                    "value": "updatedtagvalue",
                }
            ]

        # Patch the Fleet, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        fleet = ec2_validator.get_fleet(fleet_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=fleet["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=fleet["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the fleet resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        fleet = ec2_validator.get_fleet(fleet_id)
        tags.assert_ack_system_tags(
            tags=fleet["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=fleet["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Attempt to Update Fleet Target Capacity
        # This will fail as fleets of type instant cannot be modified after creation
        initialFleetTargetCapacity = fleet['TargetCapacitySpecification']['TotalTargetCapacity']
        updatedFleetTargetCapacity = 2
        updates = {
            "spec": {
                "targetCapacitySpecification": {
                    "totalTargetCapacity": updatedFleetTargetCapacity,
                    "spotTargetCapacity": updatedFleetTargetCapacity
                }
            }
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.Terminal", "True", wait_periods=5)

        # Check Fleet updated value
        fleet = ec2_validator.get_fleet(fleet_id)
        assert fleet is not None
        assert fleet['TargetCapacitySpecification']['TotalTargetCapacity'] == initialFleetTargetCapacity

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Wait for AWS to start deleting resource, which can take a few minutes so no need to wait for deletion to complete
        ec2_validator.wait_fleet_state(
            fleet_id,
            'deleted_terminating',
            max_wait_seconds=180
        )

    @pytest.mark.resource_data({'fleet_type': 'maintain'})
    def test_crud_maintain(self, simple_fleet, ec2_validator):
        """
        Test creation, update and deletion of an Fleet of type maintain.
        """
        (ref, cr) = simple_fleet

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check that the resource was created
        assert cr is not None
        assert 'status' in cr
        assert 'fleetID' in cr['status']

        fleet_id = cr['status']['fleetID']
        assert fleet_id is not None
        assert fleet_id.startswith('fleet-')

        # Check Fleet exists
        fleet = ec2_validator.get_fleet(fleet_id)
        assert fleet is not None

        # Wait for AWS to complete creation
        state_reached = ec2_validator.wait_fleet_state(
            fleet_id,
            'active',
            max_wait_seconds=180
        )
        assert state_reached, f"Fleet {fleet_id} did not reach active state within timeout"

        # Wait for K8s controller to sync the state from AWS
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=30), \
            "Resource did not sync within timeout"

        # Verify state
        cr = k8s.get_resource(ref)
        assert cr['status'].get('fleetState') == 'active', \
            f"Expected fleetState active, got {cr['status'].get('fleetState')}"
        

        # Update Fleet Target Capacity
        updatedFleetTargetCapacity = 2
        updates = {
            "spec": {
                "targetCapacitySpecification": {
                    "totalTargetCapacity": updatedFleetTargetCapacity,
                    "spotTargetCapacity": updatedFleetTargetCapacity
                }
            }
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=10)

        # Check Fleet updated value
        fleet = ec2_validator.get_fleet(fleet_id)
        assert fleet is not None
        assert fleet['TargetCapacitySpecification']['TotalTargetCapacity'] == updatedFleetTargetCapacity

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        # Wait for AWS to start deleting resource, which can take a few minutes so no need to wait for deletion to complete
        ec2_validator.wait_fleet_state(
            fleet_id,
            'deleted_terminating',
            max_wait_seconds=180
        )


    @pytest.mark.resource_data({'fleet_type': 'maintain'})
    def test_crud_immutable_fields(self, simple_fleet, ec2_validator):
        """
        Spin up a maintain Fleet and attempt to update immutable fields on the Fleet, 
        Validate that these updates are blocked and result in terminal conditions, and that the AWS resource is not updated with the invalid values.
        """
        (ref, cr) = simple_fleet

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check that the resource was created
        assert cr is not None
        assert 'status' in cr
        assert 'fleetID' in cr['status']

        fleet_id = cr['status']['fleetID']
        assert fleet_id is not None
        assert fleet_id.startswith('fleet-')

        # Check Fleet exists
        fleet = ec2_validator.get_fleet(fleet_id)
        assert fleet is not None

        # Wait for AWS to complete creation
        state_reached = ec2_validator.wait_fleet_state(
            fleet_id,
            'active',
            max_wait_seconds=180
        )
        assert state_reached, f"Fleet {fleet_id} did not reach active state within timeout"

        # Wait for K8s controller to sync the state from AWS
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=30), \
            "Resource did not sync within timeout"

        # Update Fleet Default Capacity Specification
        # updates on this field are not supported, so this should not result in any updates on AWS resource
        initialDefaultTargetCapacityType = fleet['TargetCapacitySpecification']['DefaultTargetCapacityType']
        updatedDefaultTargetCapacityType = "on-demand"
        updates = {
            "spec": {
                "targetCapacitySpecification": {
                    "defaultTargetCapacityType": updatedDefaultTargetCapacityType,
                }
            }
        }
        k8s.patch_custom_resource(ref, updates)

        # Check resource prevents this invalid update and enters terminal state
        assert k8s.wait_on_condition(ref, "ACK.Terminal", "True", wait_periods=5)

        # Check fleet value has not been updated on AWS
        fleet = ec2_validator.get_fleet(fleet_id)
        assert fleet is not None
        assert fleet['TargetCapacitySpecification']['DefaultTargetCapacityType'] == initialDefaultTargetCapacityType

        # Update Fleet Default Capacity Specification back to original value
        updates = {
            "spec": {
                "targetCapacitySpecification": {
                    "defaultTargetCapacityType": initialDefaultTargetCapacityType,
                }
            }
        }
        k8s.patch_custom_resource(ref, updates)
        # Check resource prevents this invalid update and enters terminal state
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Update Fleet ReplaceUnhealthyInstances
        # updates on this field are not supported, so this should not result in any updates on AWS resource
        initialReplaceUnhealthyInstances = fleet['ReplaceUnhealthyInstances']
        updatedReplaceUnhealthyInstances = not fleet['ReplaceUnhealthyInstances']
        updates = {
            "spec": {
                "replaceUnhealthyInstances": updatedReplaceUnhealthyInstances,
            }
        }
        k8s.patch_custom_resource(ref, updates)

        # Check resource prevents this invalid update and enters terminal state
        assert k8s.wait_on_condition(ref, "ACK.Terminal", "True", wait_periods=5)

        # Check fleet value has not been updated on AWS
        fleet = ec2_validator.get_fleet(fleet_id)
        assert fleet is not None
        assert fleet['ReplaceUnhealthyInstances'] == initialReplaceUnhealthyInstances

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        # Wait for AWS to start deleting resource, which can take a few minutes so no need to wait for deletion to complete
        ec2_validator.wait_fleet_state(
            fleet_id,
            'deleted_terminating',
            max_wait_seconds=180
        )