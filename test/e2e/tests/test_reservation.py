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

"""Integration tests for the Reservations (RunInstances) API.
"""

import datetime
import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "reservations"
# highly available instance type for deterministic testing
INSTANCE_TYPE = "m4.large"
INSTANCE_COUNT = "2"
INSTANCE_AMI = "Amazon Linux 2 Kernel"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
TIMEOUT_SECONDS = 300

def get_reservation(ec2_client, reservation_id: str) -> dict:
    try:
        resp = ec2_client.describe_instances(
            Filters=[{"Name": "reservation-id", "Values": [reservation_id]}]
        )
    except Exception as e:
        logging.debug(e)
        return None
    if len(resp["Reservations"]) == 0:
        return None
    return resp["Reservations"][0]

def get_instance_states(ec2_client, reservation_id):
    instance_states = []
    try:
        reservation = get_reservation(ec2_client, reservation_id)
        for instance in reservation['Instances']:
            instance_states.append(instance['State']['Name'])
    except Exception as e:
        logging.debug(e)
    return instance_states

def wait_for_instances_or_die(ec2_client, reservation_id, desired_state, timeout_sec):
    while True:
        now = datetime.datetime.now()
        timeout = now + datetime.timedelta(seconds=timeout_sec)
        if datetime.datetime.now() >= timeout:
            pytest.fail(f"Timed out waiting for Instance to enter {desired_state} state")
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        instance_states = get_instance_states(ec2_client, reservation_id)
        instances_match_state = True
        for state in instance_states:
            if state != desired_state:
                instances_match_state = False
        if instances_match_state:
            break

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
                if INSTANCE_AMI in image['Description']:
                    return image['ImageId']
    except Exception as e:
        logging.debug(e)

@service_marker
@pytest.mark.canary
class TestReservation:
    def test_create_delete(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("reservation-ack-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        subnet_id = test_vpc.public_subnets.subnet_ids[0]
        
        ami_id = get_ami_id(ec2_client)
        test_resource_values["RESERVATION_NAME"] = resource_name
        test_resource_values["RESERVATION_MIN"] = INSTANCE_COUNT
        test_resource_values["RESERVATION_MAX"] = INSTANCE_COUNT
        test_resource_values["RESERVATION_AMI_ID"] = ami_id
        test_resource_values["RESERVATION_INSTANCE_TYPE"] = INSTANCE_TYPE
        test_resource_values["RESERVATION_SUBNET_ID"] = subnet_id

        # Load Reservation CR
        resource_data = load_ec2_resource(
            "reservation",
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
        resource_id = resource["status"]["reservationID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Reservation exists
        assert get_reservation(ec2_client, resource_id) is not None

        # Give time for instances to come up
        wait_for_instances_or_die(ec2_client, resource_id, 'running', TIMEOUT_SECONDS)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        # Reservation still exists, but instances will commence termination
        # State needs to be 'terminated' in order to remove the dependency on the shared subnet
        # for successful test cleanup
        wait_for_instances_or_die(ec2_client, resource_id, 'terminated', TIMEOUT_SECONDS)