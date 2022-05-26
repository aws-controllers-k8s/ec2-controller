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

"""Integration tests for Instance API.
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

RESOURCE_PLURAL = "instances"
# highly available instance type for deterministic testing
INSTANCE_TYPE = "m4.large"
INSTANCE_AMI = "Amazon Linux 2 Kernel"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
TIMEOUT_SECONDS = 300

def get_instance(ec2_client, instance_id: str) -> dict:
    instance = None
    try:
        resp = ec2_client.describe_instances(
            InstanceIds=[instance_id]
        )
        instance = resp["Reservations"][0]["Instances"][0]
    except Exception as e:
        logging.debug(e)
    finally:
        return instance

def get_instance_state(ec2_client, instance_id):
    instance_state = None
    try:
        instance = get_instance(ec2_client, instance_id)
        instance_state = instance["State"]["Name"]
    except Exception as e:
        logging.debug(e)
    finally:
        return instance_state

def wait_for_instance_or_die(ec2_client, instance_id, desired_state, timeout_sec):
    while True:
        now = datetime.datetime.now()
        timeout = now + datetime.timedelta(seconds=timeout_sec)
        if datetime.datetime.now() >= timeout:
            pytest.fail(f"Timed out waiting for Instance to enter {desired_state} state")
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        instance_state = get_instance_state(ec2_client, instance_id)
        if instance_state == desired_state:
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


@pytest.fixture
def instance(ec2_client):
    test_resource_values = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("instance-ack-test", 24)
    test_vpc = get_bootstrap_resources().SharedTestVPC
    subnet_id = test_vpc.public_subnets.subnet_ids[0]
        
    ami_id = get_ami_id(ec2_client)
    test_resource_values["INSTANCE_NAME"] = resource_name
    test_resource_values["INSTANCE_AMI_ID"] = ami_id
    test_resource_values["INSTANCE_TYPE"] = INSTANCE_TYPE
    test_resource_values["INSTANCE_SUBNET_ID"] = subnet_id

    # Load Instance CR
    resource_data = load_ec2_resource(
        "instance",
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
    
    yield (ref, cr)

    # Delete the instance when tests complete
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestInstance:
    def test_create_delete(self, ec2_client, instance):
        (ref, cr) = instance
        resource_id = cr["status"]["instanceID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Instance exists
        assert get_instance(ec2_client, resource_id) is not None

        # Give time for instance to come up
        wait_for_instance_or_die(ec2_client, resource_id, 'running', TIMEOUT_SECONDS)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        # Reservation still exists, but instance will commence termination
        # State needs to be 'terminated' in order to remove the dependency on the shared subnet
        # for successful test cleanup
        wait_for_instance_or_die(ec2_client, resource_id, 'terminated', TIMEOUT_SECONDS)