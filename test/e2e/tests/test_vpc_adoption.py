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

"""Integration tests for the Vpc Adoption API.
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

VPC_RESOURCE_PLURAL = "vpcs"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def vpc_adoption(request):
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("vpc-adoption", 32)
    vpc_id = get_bootstrap_resources().AdoptedVPC.vpc_id
    replacements["VPC_ADOPTION_NAME"] = resource_name
    replacements["ADOPTION_POLICY"] = "adopt"
    replacements["ADOPTION_FIELDS"] = f"{{\\\"vpcID\\\": \\\"{vpc_id}\\\"}}"

    resource_data = load_ec2_resource(
        "vpc_adoption",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, VPC_RESOURCE_PLURAL,
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
class TestVpcAdoption:
    def test_vpc_adopt_update(self, ec2_client, vpc_adoption):
        (ref, cr) = vpc_adoption

        assert cr is not None
        assert 'status' in cr
        assert 'vpcID' in cr['status']
        resource_id = cr['status']['vpcID']

        assert 'spec' in cr
        assert 'tags' in cr['spec']

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        vpc = ec2_validator.get_vpc(resource_id)
        assert len(vpc['CidrBlockAssociationSet']) == 1
        primary_cidr = vpc['CidrBlockAssociationSet'][0]['CidrBlock']
        secondary_cidr = "10.2.0.0/16"
        updates = {
            "spec": {"cidrBlocks": [primary_cidr, secondary_cidr]}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)
    
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        
        vpc = ec2_validator.get_vpc(resource_id)
        assert len(vpc['CidrBlockAssociationSet']) == 2
        assert vpc['CidrBlockAssociationSet'][0]['CidrBlock'] == primary_cidr
        assert vpc['CidrBlockAssociationSet'][1]['CidrBlock'] == secondary_cidr      
