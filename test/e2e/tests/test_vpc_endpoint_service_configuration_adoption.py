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

"""Integration tests for the Vpc Endpoint Service Configuraion Adoption.
"""

# Default to us-west-2 since that's where prow is deployed
import logging
from os import environ
import time

import pytest

from e2e import service_marker
from e2e import CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.replacement_values import REPLACEMENT_VALUES
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from acktest import tags

from e2e.tests.helper import EC2Validator


REGION = "us-west-2" if environ.get('AWS_DEFAULT_REGION') is None else environ.get('AWS_DEFAULT_REGION')
RESOURCE_PLURAL = "vpcendpointserviceconfigurations"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

@pytest.fixture
def vpc_endpoint_service_adoption():
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("vpc-es-adoption", 24)
    service_id = get_bootstrap_resources().AdoptedVpcEndpointService.service_id
    assert service_id is not None

    replacements["VPC_ENDPOINT_SERVICE_ADOPTED_NAME"] = resource_name
    replacements["ADOPTION_POLICY"] = "adopt"
    replacements["ADOPTION_FIELDS"] = f"{{\\\"serviceID\\\": \\\"{service_id}\\\"}}"

    resource_data = load_ec2_resource(
        "vpc_endpoint_service_adoption",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

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

    _, deleted = k8s.delete_custom_resource(ref, DELETE_WAIT_AFTER_SECONDS)
    assert deleted

@service_marker
@pytest.mark.canary
class TestVpcAdoption:

    def test_vpc_endpoint_service_configuration_adopt_update(self, ec2_client, vpc_endpoint_service_adoption):
        (ref, cr) = vpc_endpoint_service_adoption

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        assert cr is not None
        assert 'status' in cr
        assert 'serviceID' in cr['status']
        resource_id = cr['status']['serviceID']

        assert 'spec' in cr
        assert 'tags' in cr['spec']

        # Check VPC Endpoint Service exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_endpoint_service_configuration(resource_id)

        updated_endpoint_service_config = ec2_validator.get_vpc_endpoint_service_configuration(resource_id)

        actual_tags = updated_endpoint_service_config['Tags']
        tags.assert_ack_system_tags(actual_tags)

        name_tag = next((tag for tag in actual_tags if tag['Key'] == 'Name'), None)
        assert name_tag is not None

        name_tag = {'key': 'Name', 'value': name_tag['Value']}
        new_tag = {'key': 'TestName', 'value': 'test-value'}
        updates = {
            "spec": {"tags": [name_tag, new_tag]}
        }
        
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
    
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        updated_endpoint_service_config = ec2_validator.get_vpc_endpoint_service_configuration(resource_id)
        assert updated_endpoint_service_config is not None
        assert 'Tags' in updated_endpoint_service_config
        
        expected_tags = [{"Key": name_tag['key'], "Value": name_tag['value']},  {"Key": new_tag['key'], "Value": new_tag['value']}]
        tags.assert_equal_without_ack_tags(
            actual=updated_endpoint_service_config['Tags'],
            expected=expected_tags,
        )





