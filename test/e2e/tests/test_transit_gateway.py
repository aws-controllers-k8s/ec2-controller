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

RESOURCE_PLURAL = "transitgateways"

CREATE_WAIT_AFTER_SECONDS = 90
DELETE_WAIT_AFTER_SECONDS = 60

@pytest.fixture(scope="module")
def ec2_client():
    return boto3.client("ec2")


def get_tgw(ec2_client, tgw_id: str) -> dict:
    try:
        resp = ec2_client.describe_transit_gateways(
            TransitGatewayIds=[tgw_id]
        )
    except Exception as e:
        logging.debug(e)
        return None

    if len(resp["TransitGateways"]) == 0:
        return None
    return resp["TransitGateways"][0]


def tgw_exists(ec2_client, tgw_id: str) -> bool:
    tgw = get_tgw(ec2_client, tgw_id)
    return tgw is not None and tgw['State'] != "deleting" and tgw['State'] != "deleted"

@service_marker
@pytest.mark.canary
class TestTGW:
    def test_create_delete(self, ec2_client):
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["transitGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check TGW exists
        exists = tgw_exists(ec2_client, resource_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check TGW doesn't exist
        exists = tgw_exists(ec2_client, resource_id)
        assert not exists