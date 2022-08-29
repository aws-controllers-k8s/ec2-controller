# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

import boto3
import pytest
import time
import logging

from acktest.aws.identity import get_region
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES

VPC_CREATE_WAIT_AFTER_SECONDS = 10
VPC_RESOURCE_PLURAL = "vpcs"

def pytest_addoption(parser):
    parser.addoption("--runslow", action="store_true", default=False, help="run slow tests")

def pytest_configure(config):
    config.addinivalue_line(
        "markers", "canary: mark test to also run in canary tests"
    )
    config.addinivalue_line(
        "markers", "service(arg): mark test associated with a given service"
    )
    config.addinivalue_line(
        "markers", "slow: mark test as slow to run"
    )
    config.addinivalue_line(
        "markers", "resource_data: mark test with data to use when creating fixture"
    )

def pytest_collection_modifyitems(config, items):
    if config.getoption("--runslow"):
        return
    skip_slow = pytest.mark.skip(reason="need --runslow option to run")
    for item in items:
        if "slow" in item.keywords:
            item.add_marker(skip_slow)


@pytest.fixture(scope="module")
def ec2_client():
    region = get_region()
    return boto3.client("ec2", region)

@pytest.fixture
def simple_vpc(request):
    resource_name = random_suffix_name("vpc-ack-test", 24)
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_NAME"] = resource_name
    replacements["CIDR_BLOCK"] = "10.0.0.0/16"
    replacements["ENABLE_DNS_SUPPORT"] = "False"
    replacements["ENABLE_DNS_HOSTNAMES"] = "False"

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'cidr_block' in data:
            replacements["CIDR_BLOCK"] = data['cidr_block']
        if 'enable_dns_support' in data:
            replacements["ENABLE_DNS_SUPPORT"] = data['enable_dns_support']
        if 'enable_dns_hostnames' in data:
            replacements["ENABLE_DNS_HOSTNAMES"] = data['enable_dns_hostnames']

    # Load VPC CR
    resource_data = load_ec2_resource(
        "vpc",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, VPC_RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(VPC_CREATE_WAIT_AFTER_SECONDS)

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