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

"""Integration tests for the Subnet Adoption API.
"""

import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

SUBNET_RESOURCE_PLURAL = "subnets"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def subnet_adoption(request):
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("subnet-adoption", 32)
    subnet_id = get_bootstrap_resources().AdoptedVPC.public_subnets.subnet_ids[0]
    replacements["SUBNET_ADOPTION_NAME"] = resource_name
    replacements["ADOPTION_POLICY"] = "adopt"
    replacements["ADOPTION_FIELDS"] = f"{{\\\"subnetID\\\": \\\"{subnet_id}\\\"}}"

    resource_data = load_ec2_resource(
        "subnet_adoption",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, SUBNET_RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    cr = k8s.wait_resource_consumed_by_controller(ref)
    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    _, deleted = k8s.delete_custom_resource(ref, DELETE_WAIT_AFTER_SECONDS)
    assert deleted


@service_marker
@pytest.mark.canary
class TestSubnetAdoption:
    def test_subnet_adopt_update(self, ec2_client, subnet_adoption):
        (ref, cr) = subnet_adoption

        assert cr is not None
        assert 'status' in cr
        assert 'subnetID' in cr['status']
        resource_id = cr['status']['subnetID']

        assert 'spec' in cr
        assert 'vpcID' in cr['spec']
        assert 'mapPublicIPOnLaunch' in cr['spec']
        mapPublicIPOnLaunch = not cr['spec']['mapPublicIPOnLaunch']
        # Check Subnet exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_subnet(resource_id)

        updates = {
            "spec": {"mapPublicIPOnLaunch": mapPublicIPOnLaunch},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)
    
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        subnet = ec2_validator.get_subnet(resource_id)
        assert subnet['MapPublicIpOnLaunch'] == mapPublicIPOnLaunch
