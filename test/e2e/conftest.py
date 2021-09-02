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

import pytest

from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from acktest.k8s import resource as k8s

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

def pytest_collection_modifyitems(config, items):
    if config.getoption("--runslow"):
        return
    skip_slow = pytest.mark.skip(reason="need --runslow option to run")
    for item in items:
        if "slow" in item.keywords:
            item.add_marker(skip_slow)


# Need to use pytest_sessionstart mechanism to store
# vpc resource information in cache (accessible to worker threads).
# This implementation is needed to share resources with a "session" scope whether pytest-xdist is used or not.
def create_vpc_resource():
    resource_name = random_suffix_name("vpc-for-tests", 24)
    test_resource_values = REPLACEMENT_VALUES.copy()
    test_resource_values["VPC_NAME"] = resource_name
    test_resource_values["CIDR_BLOCK"] = "10.0.0.0/16"

    resource_data = load_ec2_resource(
        "vpc",
        additional_replacements=test_resource_values,
    )
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, "vpcs",
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    resource = k8s.get_resource(ref)
    test_resource_values["VPC_ID"] = resource["status"]["vpcID"]

    return ref, cr


def pytest_sessionstart(session):
    worker_input = getattr(session.config, 'workerinput', None)
    if worker_input is None:
        # Create vpc when main thread enters; don't for worker threads
        ref, cr = create_vpc_resource()
        vpc_id = cr['status']['vpcID']
        vpc_cidr = cr['spec']['cidrBlock']

        session.config.cache.set("vpc_id", vpc_id)
        session.config.cache.set("vpc_cidr", vpc_cidr)
        session.config.cache.set("resource_name", ref.name)


def pytest_sessionfinish(session):
    worker_input = getattr(session.config, 'workerinput', None)
    if worker_input is None:
        # Main thread enters after workers and deletes VPC
        vpc_resource_name = session.config.cache.get("resource_name", None)
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, "vpcs",
            vpc_resource_name, namespace="default",
        )
        k8s.delete_custom_resource(ref)