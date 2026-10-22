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
IgnoreFieldDrift feature gate is Alpha and disabled by default, so the tests
enable it on the deployed controller. EC2 is one of the controllers the runtime
presubmit regenerates and e2e-tests, so this coverage runs on runtime PRs.

Enabling the gate restarts the shared controller Deployment, which is a
cluster-wide side effect in a suite that runs 32 xdist workers in parallel: a
restart mid-run can time out any other worker waiting on ACK.ResourceSynced.
The gate is therefore turned on exactly once per run and never turned back
off -- see ignore_field_drift_enabled for why that is both safe and necessary.
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
DELETE_WAIT_AFTER_SECONDS = 10

# Controller deployment coordinates in the kind test cluster (see
# test-infra/scripts/controller-setup.sh and run-e2e-tests.sh).
CONTROLLER_NAMESPACE = "ack-system"
CONTROLLER_DEPLOYMENT = "ack-ec2-controller"
CONTROLLER_CONTAINER = "controller"
FEATURE_GATE = "IgnoreFieldDrift"
# Generous window for the new pod to roll out and take over reconciliation.
ROLLOUT_WAIT_SECONDS = 120

# How long to wait for a resource to reach ACK.ResourceSynced=True (120s).
SYNC_WAIT_PERIODS = 12
SYNC_PERIOD_LENGTH = 10

# An inert annotation patched onto the CR purely to trigger a reconcile. An
# out-of-band AWS change produces no watch event, and this controller's resync
# period is the runtime default of 10 hours -- config/controller/deployment.yaml
# (what the e2e job deploys via kustomize) passes no --reconcile-*-resync-seconds
# override and VPC's RequeueOnSuccessSeconds() is 0, so getResyncPeriod falls
# through to defaultResyncPeriod. Without an explicit nudge the controller would
# not look at the resource again for the rest of the run, and any assertion about
# what it did with the drift would be vacuous -- it would hold identically with
# the feature removed.
#
# Touching an annotation suffices because the runtime adds
# AnnotationChangedPredicate to the event filter whenever the IgnoreFieldDrift
# gate is on (runtime reconciler.go, SetupWithManager); the default filter is
# GenerationChangedPredicate alone, which an annotation edit would not satisfy.
# Keeping the probe off the spec means the only delta the reconcile sees is the
# external drift itself.
RECONCILE_PROBE_ANNOTATION = "e2e.test.ack.aws.dev/reconcile-probe"


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


def _gate_enabled_in_spec() -> bool:
    """Returns True if the controller Deployment's FEATURE_GATES env var already
    has FEATURE_GATE turned on. Read from the Deployment spec (not the running
    pod), so it reflects a patch that has been accepted but is still rolling."""
    pairs = _parse_gates(_get_feature_gates_env())
    return pairs.get(FEATURE_GATE) == "true"


def _parse_gates(existing: str) -> dict:
    """Parses a FEATURE_GATES string ("A=true,B=false") into a dict."""
    pairs = {}
    for part in filter(None, (p.strip() for p in existing.split(","))):
        if "=" in part:
            k, v = part.split("=", 1)
            pairs[k.strip()] = v.strip()
    return pairs


def _merge_gate(existing: str, gate: str, enabled: bool) -> str:
    """Returns a FEATURE_GATES string with `gate` set to `enabled`, preserving
    any other gates already present."""
    pairs = _parse_gates(existing)
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


@pytest.fixture(scope="session")
def ignore_field_drift_enabled():
    """Turns the IgnoreFieldDrift feature gate on for the run, exactly once, and
    deliberately never turns it back off.

    Session scope is per WORKER, not per run: pytest-xdist distributes
    individual tests across 32 worker processes, so a fixture at any scope is
    instantiated once in every worker that happens to pick up a test from this
    file. A module-scoped enable/restore pair therefore rolled the shared
    controller Deployment up to six times in one run (three workers x
    enable + restore), and each restart could time out an unrelated worker
    waiting on ACK.ResourceSynced.

    Two properties keep this to a single restart:

      - Check-then-set. A worker that finds the gate already on in the
        Deployment spec skips the patch. Concurrent workers that both observe it
        off compute the same FEATURE_GATES string from the same starting value,
        so the second patch leaves the pod template byte-identical, does not bump
        the Deployment generation, and does not trigger a second rollout. That
        makes an explicit cross-process lock unnecessary.

      - No restore. Restoring would cost a second rollout, and a teardown in one
        worker would disable the gate underneath a drift test still running in
        another. Leaving it on is safe because the gate is inert unless a
        resource carries the ignore-field-drift annotation, which only this
        file's resources do; every other test in the suite behaves identically
        with it on. The kind cluster is torn down at the end of the run, so
        nothing outlives it.

    One rollout mid-run is still a shared-cluster hiccup. Removing it entirely
    means enabling the gate at controller setup time (FEATURE_GATES in
    test-infra's controller-setup.sh, as IAMRoleSelector already does), after
    which this fixture becomes a no-op check.
    """
    if not _gate_enabled_in_spec():
        _set_feature_gates_env(_merge_gate(_get_feature_gates_env(), FEATURE_GATE, True))
    # Whether we patched or another worker did, do not hand out the fixture
    # until the controller serving the gate is actually up.
    _wait_for_rollout()
    yield


@pytest.fixture
def ignore_field_drift_vpc(request):
    """A VPC annotated with ignore-field-drift, carrying a declared tag
    (team=payments) and DNS attributes as the create-time baseline.

    Parametrize the ignored paths (and optionally the baseline DNS-support
    value) via indirect fixture params, e.g.:

        @pytest.mark.parametrize(
            "ignore_field_drift_vpc",
            [{"ignore_paths": "spec.enableDNSSupport"}],
            indirect=True,
        )

    Defaults to ignoring spec.tags so callers that don't parametrize keep the
    original behaviour."""
    param = getattr(request, "param", None) or {}
    ignore_paths = param.get("ignore_paths", "spec.tags")
    enable_dns_support = param.get("enable_dns_support", "False")

    resource_name = random_suffix_name("vpc-ifd-test", 24)
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_NAME"] = resource_name
    replacements["CIDR_BLOCK"] = PRIMARY_CIDR_DEFAULT
    replacements["IGNORE_PATHS"] = ignore_paths
    replacements["ENABLE_DNS_SUPPORT"] = enable_dns_support
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

    # wait_resource_consumed_by_controller returns as soon as the resource has
    # any .status at all, which is the first status write and can predate
    # status.vpcID. Every test below reads vpcID out of the CR this fixture
    # yields, so a snapshot taken at that moment gives them a KeyError they
    # cannot recover from. Wait for a real reconcile, then re-read.
    assert k8s.wait_on_condition(
        ref, "ACK.ResourceSynced", "True",
        wait_periods=SYNC_WAIT_PERIODS, period_length=SYNC_PERIOD_LENGTH,
    ), f"VPC {resource_name} never reached ACK.ResourceSynced=True"
    cr = k8s.get_resource(ref)
    assert cr["status"]["vpcID"]

    yield (ref, cr)

    # A teardown failure must not be silent: a swallowed assertion here means a
    # leaked VPC or CR passes unnoticed, which is exactly the class of leak that
    # exhausts the bootstrap cleanup retries later in the run.
    #
    # 60s rather than the more common 30s: a VPC CR whose delete is queued behind
    # a busy controller routinely needs more than 30s to clear its finalizer, and
    # timing out here would report a leak that is merely slow.
    _, deleted = k8s.delete_custom_resource(ref, 6, 10)
    assert deleted, f"CR {ref.name} was not deleted within 60s; possible leak"


def _user_tags(vpc: dict) -> dict:
    """Returns the VPC's tags as a {key: value} dict, excluding ACK system
    tags, from a describe_vpcs response entry."""
    return tags.to_dict(
        vpc.get("Tags", []),
        key_member_name="Key",
        value_member_name="Value",
    )


def _dns_support_enabled(ec2_client, vpc_id: str) -> bool:
    """Returns the live enableDnsSupport attribute value for the VPC."""
    resp = ec2_client.describe_vpc_attribute(
        VpcId=vpc_id, Attribute="enableDnsSupport",
    )
    return resp["EnableDnsSupport"]["Value"]


def _await_reconcile_after(ref, synced_before, what: str):
    """Blocks until a reconcile that STARTED after `synced_before` has completed
    with ACK.ResourceSynced=True.

    ACK rewrites ACK.ResourceSynced.lastTransitionTime on every reconcile, so a
    strictly newer timestamp is proof the controller looked at the resource again
    rather than the test reading a condition left over from an earlier reconcile.
    A plain wait_on_condition cannot make that distinction: it returns on its
    first poll off whatever is already there, which after create is a stale
    Synced=True, so it passes without the controller having done anything.

    Doubles as the "stays Synced despite the drift" assertion -- it requires the
    fresh reconcile to have concluded Synced=True.
    """
    assert k8s.wait_on_condition_after(
        ref, "ACK.ResourceSynced", "True",
        last_transition_after=synced_before,
        wait_periods=SYNC_WAIT_PERIODS, period_length=SYNC_PERIOD_LENGTH,
    ), f"no reconcile completed with ACK.ResourceSynced=True after {what}"


def _force_reconcile_after(ref, synced_before, what: str):
    """Triggers a reconcile via the inert probe annotation, then waits for it.

    For drift applied out-of-band on the AWS side, which generates no watch
    event -- see RECONCILE_PROBE_ANNOTATION. Not needed after patching .spec,
    which bumps metadata.generation and queues a reconcile on its own; use
    _await_reconcile_after directly there.

    The probe value is a timestamp rather than a constant so that calling this
    twice within one test really changes the annotation. Re-patching an identical
    value would leave the object unchanged, fire no event, and silently reduce
    the wait below to a no-op against the previous reconcile.
    """
    k8s.patch_custom_resource(
        ref,
        {"metadata": {"annotations": {RECONCILE_PROBE_ANNOTATION: str(time.time())}}},
    )
    _await_reconcile_after(ref, synced_before, what)


@service_marker
class TestVpcIgnoreFieldDrift:
    """Verifies the services.k8s.aws/ignore-field-drift annotation on an EC2
    VPC across the field shapes and drift sources that matter:

    - test_tags_drift_ignored: an ignored list-of-objects field (spec.tags) --
      an externally-ADDED element survives, and an edit to the declared element
      is retained without being pushed.
    - test_scalar_external_drift_ignored: an ignored scalar leaf
      (spec.enableDNSSupport) whose Delta path matches the ignored path exactly
      -- an external flip survives.
    - test_scalar_spec_edit_not_pushed: the same scalar, but the edit is made
      while AWS still holds a different value, so declining to push it is
      observable. Split from the test above because the field is boolean: after
      an external flip, spec and AWS agree and the assertion would be vacuous.
    - test_declared_tag_value_drift_ignored: an externally-changed VALUE on a
      declared tag key -- the case tags.Sync would actively revert.

    In every case the controller still applies the declared value at create but
    stops reconciling drift on the ignored path: external changes survive, the
    resource stays Synced, and an edit to the ignored field is retained in the
    spec but not pushed to AWS. This mirrors the iam-controller Role coverage
    for the same runtime feature (community#2367).

    None of these exercise the runtime's path-PREFIX match: every Delta path here
    equals its annotation path. Prefix matching needs a delta subject deeper than
    the annotation, which none of VPC's ignorable fields produce."""

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

        synced_before = condition.get_synced_last_transition_time(ref)
        assert synced_before is not None

        # An external actor adds a tag ACK does not know about (the dynamic /
        # SCP-managed tag from the motivating use case).
        ec2_client.create_tags(
            Resources=[vpc_id],
            Tags=[{"Key": "external", "Value": "managed-elsewhere"}],
        )
        # finally, not a trailing cleanup call: an assertion failure below would
        # otherwise skip it and leave the out-of-band tag behind.
        try:
            # Precondition: AWS really reports the added tag before the controller
            # is asked to look. If the forced reconcile raced ahead of CreateTags,
            # sdkFind would see the original tag set, find no drift, and the
            # assertion below would pass for an unrelated reason.
            assert _user_tags(
                ec2_validator.get_vpc(vpc_id)
            ).get("external") == "managed-elsewhere", (
                "out-of-band CreateTags did not take effect"
            )

            # Nothing else will make the controller look (10h resync, no watch
            # event). This also asserts the resource stays Synced even though
            # spec.tags (team) differs from the live set (team + external).
            _force_reconcile_after(
                ref, synced_before, "the out-of-band tag was added",
            )

            # Only now is this meaningful: the controller examined the resource
            # and declined to call DeleteTags for the externally-added tag.
            tag_map = _user_tags(ec2_validator.get_vpc(vpc_id))
            assert tag_map.get("external") == "managed-elsewhere", (
                "controller removed an externally-managed tag despite "
                "ignore-field-drift on spec.tags"
            )
            assert tag_map.get("team") == "payments"

            # Editing spec.tags while ignored is retained in the spec but NOT
            # pushed to AWS: patch the CR to a different tag value and confirm the
            # live tags are unchanged.
            synced_before_edit = condition.get_synced_last_transition_time(ref)
            assert synced_before_edit is not None
            updates = {"spec": {"tags": [{"key": "team", "value": "changed"}]}}
            k8s.patch_custom_resource(ref, updates)
            # This patch bumps metadata.generation, so a reconcile is queued --
            # but waiting on a fixed sleep would not establish it had finished.
            _await_reconcile_after(
                ref, synced_before_edit, "the spec edit to the ignored field",
            )

            tag_map = _user_tags(ec2_validator.get_vpc(vpc_id))
            # external tag still present (never removed)...
            assert tag_map.get("external") == "managed-elsewhere"
            # ...and the edited value was NOT pushed (team still "payments").
            assert tag_map.get("team") == "payments"

            # The declared value is retained in the CR spec (retain semantics).
            latest = k8s.get_resource(ref)
            spec_tags = {t["key"]: t["value"] for t in latest["spec"].get("tags", [])}
            assert spec_tags.get("team") == "changed"
        finally:
            ec2_client.delete_tags(
                Resources=[vpc_id],
                Tags=[{"Key": "external"}],
            )

    @pytest.mark.parametrize(
        "ignore_field_drift_vpc",
        [{"ignore_paths": "spec.enableDNSSupport", "enable_dns_support": "False"}],
        indirect=True,
    )
    def test_scalar_external_drift_ignored(
        self, ec2_client, ignore_field_drift_enabled, ignore_field_drift_vpc,
    ):
        """Ignored scalar leaf: spec.enableDNSSupport. Its Delta path
        (Spec.EnableDNSSupport) equals the ignored path exactly, so this
        exercises the exact-match branch of the runtime's path filtering.

        This half covers EXTERNAL drift. The spec-edit half is a separate test:
        the field is boolean, so once the external actor has flipped the live
        value to True a subsequent spec edit to True makes spec and AWS agree,
        leaving no delta to suppress and nothing to assert.
        """
        (ref, cr) = ignore_field_drift_vpc
        vpc_id = cr["status"]["vpcID"]
        ec2_validator = EC2Validator(ec2_client)

        # Baseline: created with enableDNSSupport=false, and the controller
        # applied that at create (AWS defaults the attribute to true).
        ec2_validator.assert_vpc(vpc_id)
        assert _dns_support_enabled(ec2_client, vpc_id) is False
        condition.assert_synced(ref)

        synced_before = condition.get_synced_last_transition_time(ref)
        assert synced_before is not None

        # An external actor flips the attribute on AWS. Without ignore-field-drift
        # the controller would reconcile it back to the spec value (false).
        ec2_client.modify_vpc_attribute(
            VpcId=vpc_id, EnableDnsSupport={"Value": True},
        )

        # Precondition: the flip is visible on AWS before the controller looks,
        # so a race with ModifyVpcAttribute cannot make this pass for the wrong
        # reason.
        assert _dns_support_enabled(ec2_client, vpc_id) is True, (
            "out-of-band ModifyVpcAttribute did not take effect"
        )

        # Nothing else will make the controller look (10h resync, no watch event).
        # This also asserts it stays Synced even though spec (false) differs from
        # the live attribute (true).
        _force_reconcile_after(
            ref, synced_before, "the external enableDNSSupport flip",
        )

        # Only now is this meaningful: the controller examined the resource and
        # declined to call ModifyVpcAttribute to undo the flip. Non-vacuous in the
        # other sense too -- spec says False, AWS says True, so only drift
        # suppression keeps it that way.
        assert _dns_support_enabled(ec2_client, vpc_id) is True, (
            "controller reverted an externally-changed enableDNSSupport despite "
            "ignore-field-drift on spec.enableDNSSupport"
        )

    @pytest.mark.parametrize(
        "ignore_field_drift_vpc",
        [{"ignore_paths": "spec.enableDNSSupport", "enable_dns_support": "False"}],
        indirect=True,
    )
    def test_scalar_spec_edit_not_pushed(
        self, ec2_client, ignore_field_drift_enabled, ignore_field_drift_vpc,
    ):
        """Editing an ignored scalar is retained in the spec but never pushed to
        AWS.

        The edit is made while the live value still DIFFERS from the edited one,
        which is what makes the assertion meaningful: the controller has a real
        delta it could act on (spec True vs AWS False) and must decline to.
        Asserting this after an external flip to True would be vacuous, since
        spec and AWS would already agree.

        Distinct from the spec.tags edit case: enableDNSSupport is pushed by a
        ModifyVpcAttribute call in the controller's custom update path, not by
        tags.Sync, so suppression has to hold for that path too.
        """
        (ref, cr) = ignore_field_drift_vpc
        vpc_id = cr["status"]["vpcID"]
        ec2_validator = EC2Validator(ec2_client)

        # Baseline: created with enableDNSSupport=false and applied at create.
        ec2_validator.assert_vpc(vpc_id)
        assert _dns_support_enabled(ec2_client, vpc_id) is False
        condition.assert_synced(ref)

        synced_before = condition.get_synced_last_transition_time(ref)
        assert synced_before is not None

        # Edit the ignored scalar to a value AWS does NOT currently hold.
        k8s.patch_custom_resource(ref, {"spec": {"enableDNSSupport": True}})

        # The patch bumps metadata.generation, so a reconcile is queued -- but a
        # fixed sleep would not establish that it had finished, and the assertion
        # below is only meaningful once it has. Also asserts the resource stays
        # Synced even though spec (true) differs from the live attribute (false).
        _await_reconcile_after(
            ref, synced_before, "the spec edit to the ignored scalar",
        )

        # The edit must not reach AWS: the field is ignored, so no
        # ModifyVpcAttribute is issued and the live value stays False.
        assert _dns_support_enabled(ec2_client, vpc_id) is False, (
            "controller pushed an edit to spec.enableDNSSupport despite "
            "ignore-field-drift on that field"
        )

        # ...and the declared value is retained in the CR spec (retain semantics).
        latest = k8s.get_resource(ref)
        assert latest["spec"].get("enableDNSSupport") is True

    @pytest.mark.parametrize(
        "ignore_field_drift_vpc",
        [{"ignore_paths": "spec.tags"}],
        indirect=True,
    )
    def test_declared_tag_value_drift_ignored(
        self, ec2_client, ignore_field_drift_enabled, ignore_field_drift_vpc,
    ):
        """Drift on the VALUE of a tag key the CR declares, rather than on an
        externally-added key.

        Materially different from test_tags_drift_ignored: an added key is one
        tags.Sync would leave alone anyway, whereas a changed value on a declared
        key is one it would actively revert, so this is the case where drift
        suppression does the real work.

        Note this does NOT exercise prefix matching. The generated delta compares
        tags as a whole (pkg/resource/vpc/delta.go emits a single
        delta.Add("Spec.Tags", ...) via MapStringStringEqual), so the Delta path
        is exactly Spec.Tags and Path.ContainsFold("spec.tags") matches by
        segment equality. Prefix matching needs a Delta path DEEPER than the
        annotation -- a nested struct member for which code-gen emits a
        Spec.Parent.Child subject -- and per-element paths like Spec.Tags.N.Value
        neither exist in the delta nor are expressible as an annotation
        (isValidFieldPath rejects indices).
        """
        (ref, cr) = ignore_field_drift_vpc
        vpc_id = cr["status"]["vpcID"]
        ec2_validator = EC2Validator(ec2_client)

        # Baseline: the declared tag was applied at create.
        ec2_validator.assert_vpc(vpc_id)
        assert _user_tags(ec2_validator.get_vpc(vpc_id)).get("team") == "payments"
        condition.assert_synced(ref)

        synced_before = condition.get_synced_last_transition_time(ref)
        assert synced_before is not None

        # An external actor overwrites the VALUE of the declared tag key, rather
        # than adding a new key.
        ec2_client.create_tags(
            Resources=[vpc_id],
            Tags=[{"Key": "team", "Value": "external-override"}],
        )

        # Precondition: the overwrite is visible on AWS before the controller
        # looks, so a race with CreateTags cannot make this pass for the wrong
        # reason.
        assert _user_tags(
            ec2_validator.get_vpc(vpc_id)
        ).get("team") == "external-override", (
            "out-of-band CreateTags did not overwrite the declared tag value"
        )

        # Nothing else will make the controller look (10h resync, no watch event).
        # This also asserts it stays Synced even though the spec tag value
        # (payments) differs from the live value (external-override).
        _force_reconcile_after(
            ref, synced_before, "the declared tag value was overwritten",
        )

        # Only now is this meaningful: the controller examined the resource and
        # declined to revert the declared key's value back to the spec value --
        # the case tags.Sync would actively undo.
        assert _user_tags(ec2_validator.get_vpc(vpc_id)).get("team") == "external-override", (
            "controller reverted an externally-changed tag value despite "
            "ignore-field-drift on spec.tags"
        )
