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

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "egressonlyinternetgateways"

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
