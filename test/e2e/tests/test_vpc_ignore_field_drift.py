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

"""Integration tests for the services.k8s.aws/ignore-field-drift annotation,
exercised against the EC2 VPC resource and its spec.tags field.

This is the runtime feature under aws-controllers-k8s/runtime#256. The
IgnoreFieldDrift feature gate is Alpha and disabled by default, so the test
enables it on the deployed controller for the duration of the module and
restores the prior value afterwards. EC2 is one of the controllers the runtime
presubmit regenerates and e2e-tests, so this coverage runs on runtime PRs.
"""

import logging
import time

import boto3
import pytest

from acktest import tags
from acktest.k8s import condition
from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "vpcs"
PRIMARY_CIDR_DEFAULT = "10.0.0.0/16"

CREATE_WAIT_AFTER_SECONDS = 15
MODIFY_WAIT_AFTER_SECONDS = 20
DELETE_WAIT_AFTER_SECONDS = 10

# Controller deployment coordinates in the kind test cluster (see
# test-infra/scripts/controller-setup.sh and run-e2e-tests.sh).
CONTROLLER_NAMESPACE = "ack-system"
CONTROLLER_DEPLOYMENT = "ack-ec2-controller"
CONTROLLER_CONTAINER = "controller"
FEATURE_GATE = "IgnoreFieldDrift"
# Generous window for the new pod to roll out and take over reconciliation.
ROLLOUT_WAIT_SECONDS = 120


def _apps_client():
    # Build the AppsV1Api against acktest's configured ApiClient (which points
    # at the kind cluster). A bare AppsV1Api() would default to localhost:80.
    from kubernetes import client as k8s_client
    return k8s_client.AppsV1Api(k8s._get_k8s_api_client())


def _get_feature_gates_env() -> str:
    """Returns the current value of the FEATURE_GATES env var on the controller
    container, or "" if it is unset."""
    dep = _apps_client().read_namespaced_deployment(
        CONTROLLER_DEPLOYMENT, CONTROLLER_NAMESPACE,
    )
    for c in dep.spec.template.spec.containers:
        if c.name != CONTROLLER_CONTAINER:
            continue
        for e in (c.env or []):
            if e.name == "FEATURE_GATES":
                return e.value or ""
    return ""


def _set_feature_gates_env(value: str):
    """Patches the FEATURE_GATES env var on the controller container and waits
    for the rollout to complete. The controller wires this env var into its
    --feature-gates flag (see the controller deployment manifest)."""
    body = {
        "spec": {
            "template": {
                "spec": {
                    "containers": [
                        {"name": CONTROLLER_CONTAINER,
                         "env": [{"name": "FEATURE_GATES", "value": value}]},
                    ]
                }
            }
        }
    }
    _apps_client().patch_namespaced_deployment(
        CONTROLLER_DEPLOYMENT, CONTROLLER_NAMESPACE, body,
    )
    _wait_for_rollout()


def _merge_gate(existing: str, gate: str, enabled: bool) -> str:
    """Returns a FEATURE_GATES string with `gate` set to `enabled`, preserving
    any other gates already present."""
    pairs = {}
    for part in filter(None, (p.strip() for p in existing.split(","))):
        if "=" in part:
            k, v = part.split("=", 1)
            pairs[k.strip()] = v.strip()
    pairs[gate] = "true" if enabled else "false"
    return ",".join(f"{k}={v}" for k, v in pairs.items())


def _wait_for_rollout():
    """Blocks until the controller deployment reports all replicas updated and
    available for the current generation."""
    client = _apps_client()
    deadline = time.time() + ROLLOUT_WAIT_SECONDS
    while time.time() < deadline:
        dep = client.read_namespaced_deployment(
            CONTROLLER_DEPLOYMENT, CONTROLLER_NAMESPACE,
        )
        spec_replicas = dep.spec.replicas or 1
        status = dep.status
        if (status.observed_generation is not None
                and status.observed_generation >= dep.metadata.generation
                and (status.updated_replicas or 0) >= spec_replicas
                and (status.available_replicas or 0) >= spec_replicas
                and (status.unavailable_replicas or 0) == 0):
            # Give the fresh pod a moment to acquire leadership / start reconciling.
            time.sleep(5)
            return
        time.sleep(3)
    raise AssertionError(
        f"controller deployment {CONTROLLER_DEPLOYMENT} did not roll out within "
        f"{ROLLOUT_WAIT_SECONDS}s after toggling the {FEATURE_GATE} feature gate"
    )


@pytest.fixture(scope="module")
def ec2_client():
    from acktest.aws.identity import get_region
    return boto3.client("ec2", get_region())


@pytest.fixture(scope="module")
def ignore_field_drift_enabled():
    """Enables the IgnoreFieldDrift feature gate on the controller for the
    duration of the module, then restores the prior FEATURE_GATES value."""
    original = _get_feature_gates_env()
    _set_feature_gates_env(_merge_gate(original, FEATURE_GATE, True))
    yield
    # Restore exactly what was there before (which may be "").
    _set_feature_gates_env(original)


@pytest.fixture
def ignore_field_drift_vpc(request):
    """A VPC annotated with ignore-field-drift on spec.tags, carrying one
    declared tag (team=payments) as the create-time baseline."""
    resource_name = random_suffix_name("vpc-ifd-test", 24)
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_NAME"] = resource_name
    replacements["CIDR_BLOCK"] = PRIMARY_CIDR_DEFAULT
    replacements["ENABLE_DNS_SUPPORT"] = "False"
    replacements["ENABLE_DNS_HOSTNAMES"] = "False"
    replacements["ENABLE_NETWORK_ADDRESS_USAGE_METRICS"] = "False"
    replacements["DISALLOW_DEFAULT_SECURITY_GROUP_RULE"] = "False"
    replacements["TAG_KEY"] = "team"
    replacements["TAG_VALUE"] = "payments"

    resource_data = load_ec2_resource(
        "vpc_ignore_field_drift",
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

    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass


def _user_tags(vpc: dict) -> dict:
    """Returns the VPC's tags as a {key: value} dict, excluding ACK system
    tags, from a describe_vpcs response entry."""
    return tags.to_dict(
        vpc.get("Tags", []),
        key_member_name="Key",
        value_member_name="Value",
    )


@service_marker
class TestVpcIgnoreFieldDrift:
    """Verifies the services.k8s.aws/ignore-field-drift annotation on an EC2
    VPC's spec.tags: the controller still applies the declared tag at create,
    but stops reconciling drift on tags -- externally-added tags survive, the
    resource stays Synced, and an edit to the ignored field is retained in the
    spec but not pushed to AWS. This mirrors the iam-controller Role coverage
    for the same runtime feature (community#2367)."""

    def test_tags_drift_ignored(
        self, ec2_client, ignore_field_drift_enabled, ignore_field_drift_vpc,
    ):
        (ref, cr) = ignore_field_drift_vpc
        vpc_id = cr["status"]["vpcID"]
        ec2_validator = EC2Validator(ec2_client)

        # Baseline: the declared tag was applied at create.
        ec2_validator.assert_vpc(vpc_id)
        vpc = ec2_validator.get_vpc(vpc_id)
        assert _user_tags(vpc).get("team") == "payments"

        # The resource is Synced after create.
        condition.assert_synced(ref)

        # An external actor adds a tag ACK does not know about (the dynamic /
        # SCP-managed tag from the motivating use case).
        ec2_client.create_tags(
            Resources=[vpc_id],
            Tags=[{"Key": "external", "Value": "managed-elsewhere"}],
        )
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # The externally-added tag must survive: ACK does not reconcile drift on
        # spec.tags, so it does not call DeleteTags for it.
        tag_map = _user_tags(ec2_validator.get_vpc(vpc_id))
        assert tag_map.get("external") == "managed-elsewhere", (
            "controller removed an externally-managed tag despite "
            "ignore-field-drift on spec.tags"
        )
        assert tag_map.get("team") == "payments"

        # The resource stays Synced even though spec.tags (team) differs from the
        # live AWS tag set (team + external).
        assert k8s.wait_on_condition(
            ref, "ACK.ResourceSynced", "True",
            wait_periods=6, period_length=10,
        )

        # Editing spec.tags while ignored is retained in the spec but NOT pushed
        # to AWS: patch the CR to a different tag value and confirm the live tags
        # are unchanged.
        updates = {"spec": {"tags": [{"key": "team", "value": "changed"}]}}
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        tag_map = _user_tags(ec2_validator.get_vpc(vpc_id))
        # external tag still present (never removed)...
        assert tag_map.get("external") == "managed-elsewhere"
        # ...and the edited value was NOT pushed (team still "payments").
        assert tag_map.get("team") == "payments"

        # The declared value is retained in the CR spec (retain semantics).
        latest = k8s.get_resource(ref)
        spec_tags = {t["key"]: t["value"] for t in latest["spec"].get("tags", [])}
        assert spec_tags.get("team") == "changed"

        # Clean up the out-of-band tag so teardown deletes cleanly.
        ec2_client.delete_tags(
            Resources=[vpc_id],
            Tags=[{"Key": "external"}],
        )
