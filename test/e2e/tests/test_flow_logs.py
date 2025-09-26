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

"""Integration tests for the Flow Log API.
"""

import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s, condition
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "flowlogs"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5


def get_flow_log_ids(ec2_client, flow_log_id: str) -> list:
    flow_log_ids = [flow_log_id]
    try:
        resp = ec2_client.describe_flow_logs(
            FlowLogIds=flow_log_ids
        )
    except Exception as e:
        logging.debug(e)
        return None

    flow_log_ids = []
    for flow_log in resp['FlowLogs']:
        flow_log_ids.append(flow_log['FlowLogId'])

    if len(flow_log_ids) == 0:
        return None
    return flow_log_ids


def flow_log_exists(ec2_client, flow_log_id: str) -> bool:
    return get_flow_log_ids(ec2_client, flow_log_id) is not None

@pytest.fixture
def simple_flow_log(request):
    resource_name = random_suffix_name("flow-log-ack-test", 24)
    resource_file = "flow_log"
    resources = get_bootstrap_resources()


    replacements = REPLACEMENT_VALUES.copy()
    replacements["FLOWLOG_NAME"] = resource_name
    replacements["RESOURCE_ID"] = resources.SharedTestVPC.vpc_id
    replacements["RESOURCE_TYPE"] = "VPC"
    replacements["LOG_DESTINATION_TYPE"] = "s3"
    replacements["LOG_DESTINATION"] = "arn:aws:s3:::" + resources.FlowLogsBucket.name 
    replacements["TRAFFIC_TYPE"] = "ALL"
    replacements["TAG_KEY"] = "Name"
    replacements["TAG_VALUE"] = resource_name

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'resource_file' in data:
            resource_file = data['resource_file']
        if 'resource_type' in data:
            replacements["RESOURCE_TYPE"] = data['resource_type']

    # Load FlowLog CR
    resource_data = load_ec2_resource(
        resource_file,
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
class TestFlowLogs:
    def test_create_delete(self, ec2_client, simple_flow_log):
        (ref, cr) = simple_flow_log
        resource_id = cr["status"]["flowLogID"]

        # Check Flow Log exists
        exists = flow_log_exists(ec2_client, resource_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Flow Log doesn't exist
        exists = flow_log_exists(ec2_client, resource_id)
        assert not exists

    @pytest.mark.resource_data({'resource_type': 'InvalidResource'})
    def test_terminal_condition_invalid_parameter_value(self, simple_flow_log):
        (ref, cr) = simple_flow_log

        expected_msg = "InvalidParameterValue: "
        condition.assert_terminal(ref, expected_msg)

    @pytest.mark.resource_data({'resource_file': 'invalid/flow_log_invalid_parameter'})
    def test_terminal_condition_invalid_parameter(self, simple_flow_log):
        (ref, cr) = simple_flow_log

        expected_msg = "InvalidParameter: "
        condition.assert_terminal(ref, expected_msg)
