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

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "transitgateways"

## The long delete wait is required to make sure the TGW can transition out of its "pending" status.
## TGWs are unable to be deleted while in "pending"
CREATE_WAIT_AFTER_SECONDS = 90
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def simple_transit_gateway():
    resource_name = random_suffix_name("tgw-ack-test", 24)
    replacements = REPLACEMENT_VALUES.copy()
    replacements["TGW_NAME"] = resource_name

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
class TestTGW:
    def test_create_delete(self, ec2_client, simple_transit_gateway):
        (ref, cr) = simple_transit_gateway
        resource_id = cr["status"]["transitGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check TGW exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_transit_gateway(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check TGW no longer exists in AWS
        ec2_validator.assert_transit_gateway(resource_id, exists=False)