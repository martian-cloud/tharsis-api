// SAMPLE_OPA_POLICY is the default Rego pre-populated in the policy editor on the
// new package version page. It denies nothing on its own — it documents the
// input document shape and shows the deny convention via commented-out examples
// so authors have a correct, self-teaching starting point. The paths must stay in
// sync with buildInputDocument in internal/jobexecutor/policyeval.go, which is
// what actually assembles the document: `input.run.*` (run metadata *and* the
// variables), `input.tfplan.*` (the plan, wrapped, and only at the post_plan
// stage), plus the top-level `input.schemaVersion` and `input.stage`.
//
// NOTE: this is a JS template literal, so the Rego examples deliberately use
// double-quoted regex strings rather than backtick raw strings (a backtick here
// would terminate the string).
export const SAMPLE_OPA_POLICY = `# Tharsis policy (Rego)
#
# A run is DENIED when this policy produces one or more "deny" (or "violation")
# messages; if no messages are produced, the run is allowed. As written this
# template denies nothing — uncomment and adapt the examples below.
#
# The input document available to every policy:
#   input.schemaVersion   Version of this input document's schema (currently 1).
#   input.stage           Stage being evaluated: "pre_plan" or "post_plan".
#   input.run             Run metadata: id, createdBy, isDestroy, refresh,
#                         refreshOnly, speculative, terraformVersion,
#                         targetAddresses, moduleSource, moduleVersion,
#                         moduleDigest, configurationVersionId, workspacePath,
#                         workspaceId.
#   input.run.variables   Run variables, most specific first:
#                         [{ key, category, sensitive, namespacePath, value }].
#                         category is "terraform" or "environment"; namespacePath
#                         is absent for run-scoped variables; value is absent when
#                         the variable has none.
#   input.tfplan          The Terraform plan JSON, e.g.
#                         input.tfplan.resource_changes.
#
# Two things worth knowing before you write a rule:
#
#   * input.tfplan exists ONLY at the post_plan stage — a pre_plan check runs
#     before there is a plan. Plan rules are simply undefined (they never fire)
#     at pre_plan, so a policy can safely be attached to both stages.
#   * The optional run fields (moduleSource, moduleVersion, moduleDigest,
#     configurationVersionId) are always present but are null when unset, so test
#     them against null rather than against ""; targetAddresses is null or empty
#     for a run with no targets. Variable values are always raw strings — either
#     the literal string, or HCL source for a list/map/number variable — so use
#     to_number, split, and friends rather than comparing to a non-string.

package tharsis.policy

import rego.v1

# --- Examples -----------------------------------------------------------------
# Each rule below is commented out. Uncomment and adapt the ones you want to
# enforce. Any rule named "deny" (or "violation") that produces a message fails
# the run; define as many as you like.

# 1) Block destroy runs entirely.
# deny contains msg if {
#     input.run.isDestroy
#     msg := "Destroy runs are not permitted in this workspace."
# }

# 2) Forbid destroying any existing resource.
# deny contains msg if {
#     resource := input.tfplan.resource_changes[_]
#     resource.change.actions[_] == "delete"
#     msg := sprintf("Resource '%v' (%v) is scheduled for destruction.", [resource.address, resource.type])
# }

# 3) Require registry modules to be pinned to a released version. moduleVersion
#    is null (not "") for a run that has no module version, hence the null test.
# deny contains msg if {
#     input.run.moduleSource != null
#     object.get(input.run, "moduleVersion", null) == null
#     msg := sprintf("Module '%v' must be pinned to a released version.", [input.run.moduleSource])
# }

# 4) Reject pre-release module versions (e.g. 1.2.0-rc1, 2.0.0-beta).
# deny contains msg if {
#     version := object.get(input.run, "moduleVersion", null)
#     version != null
#     contains(version, "-")
#     msg := sprintf("Module version '%v' is a pre-release; use a released version.", [version])
# }

# 5) Require secret-looking variables to be marked sensitive.
# deny contains msg if {
#     variable := input.run.variables[_]
#     regex.match("(?i)(password|secret|token|api[_-]?key)", variable.key)
#     not variable.sensitive
#     msg := sprintf("Variable '%v' looks like a secret but is not marked sensitive.", [variable.key])
# }

# 6) Disallow public-read S3 buckets.
# deny contains msg if {
#     resource := input.tfplan.resource_changes[_]
#     resource.type == "aws_s3_bucket"
#     resource.change.after.acl == "public-read"
#     msg := sprintf("S3 bucket '%v' must not be public-read.", [resource.address])
# }

# 7) Require an "Environment" tag on newly created resources.
# deny contains msg if {
#     resource := input.tfplan.resource_changes[_]
#     resource.change.actions[_] == "create"
#     not resource.change.after.tags.Environment
#     msg := sprintf("Resource '%v' must have an 'Environment' tag.", [resource.address])
# }

# 8) Cap the number of resources a single plan may change.
# deny contains msg if {
#     limit := 100
#     count(input.tfplan.resource_changes) > limit
#     msg := sprintf("Plan changes %v resources, which exceeds the limit of %v.", [count(input.tfplan.resource_changes), limit])
# }

# 9) Forbid targeted runs, checked at pre_plan so it fails before planning
#    rather than after. Gating on input.stage is what you want for a rule that
#    needs no plan; the null guard is the optional-field caveat above.
# deny contains msg if {
#     input.stage == "pre_plan"
#     targets := object.get(input.run, "targetAddresses", null)
#     targets != null
#     count(targets) > 0
#     msg := sprintf("Targeted runs are not permitted; %v address(es) were targeted.", [count(targets)])
# }
`;


// SAMPLE_OPA_INPUTS are example input documents a Tharsis OPA policy is evaluated
// against at run time. They are shown (read-only) from the new package version
// page so authors can see the exact shape their Rego rules reference. Each one
// mirrors buildInputDocument in internal/jobexecutor/policyeval.go field for
// field, including the parts that trip authors up: variables nested under
// `run`, optional run fields present as null, string-valued variables, and a
// pre_plan document with no `tfplan` key at all.
export const SAMPLE_OPA_INPUTS: { id: string; label: string; content: string }[] = [
    {
        id: 'post-plan-create',
        label: 'Post-plan — create S3 bucket',
        content: JSON.stringify({
            schemaVersion: 1,
            stage: 'post_plan',
            run: {
                id: 'run-2',
                createdBy: 'user@example.com',
                isDestroy: false,
                refresh: true,
                refreshOnly: false,
                speculative: false,
                terraformVersion: '1.9.8',
                targetAddresses: null,
                moduleSource: 'registry.terraform.io/example/s3/aws',
                moduleVersion: '1.0.0',
                moduleDigest: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
                configurationVersionId: null,
                workspacePath: 'my-group/prod',
                workspaceId: 'ws-2',
                variables: [
                    { key: 'environment', category: 'terraform', sensitive: false, namespacePath: 'my-group/prod', value: 'prod' },
                    { key: 'availability_zones', category: 'terraform', sensitive: false, namespacePath: 'my-group', value: '["us-east-1a", "us-east-1b"]' },
                    { key: 'AWS_REGION', category: 'environment', sensitive: false, value: 'us-east-1' },
                    { key: 'db_password', category: 'terraform', sensitive: true, namespacePath: 'my-group/prod', value: 'not-the-real-value' },
                ],
            },
            tfplan: {
                format_version: '1.2',
                terraform_version: '1.9.8',
                resource_changes: [
                    {
                        address: 'aws_s3_bucket.data',
                        type: 'aws_s3_bucket',
                        name: 'data',
                        provider_name: 'registry.terraform.io/hashicorp/aws',
                        change: { actions: ['create'], before: null, after: { bucket: 'my-data-bucket', acl: 'public-read', tags: {} } },
                    },
                ],
            },
        }, null, 2),
    },
    {
        id: 'post-plan-destroy',
        label: 'Post-plan — destroy run',
        content: JSON.stringify({
            schemaVersion: 1,
            stage: 'post_plan',
            run: {
                id: 'run-3',
                createdBy: 'user@example.com',
                isDestroy: true,
                refresh: true,
                refreshOnly: false,
                speculative: false,
                terraformVersion: '1.9.8',
                targetAddresses: null,
                moduleSource: null,
                moduleVersion: null,
                moduleDigest: null,
                configurationVersionId: 'cv-3',
                workspacePath: 'my-group/prod',
                workspaceId: 'ws-2',
                variables: [],
            },
            tfplan: {
                format_version: '1.2',
                terraform_version: '1.9.8',
                resource_changes: [
                    {
                        address: 'aws_db_instance.main',
                        type: 'aws_db_instance',
                        name: 'main',
                        provider_name: 'registry.terraform.io/hashicorp/aws',
                        change: { actions: ['delete'], before: { identifier: 'prod-db' }, after: null },
                    },
                ],
            },
        }, null, 2),
    },
    {
        id: 'pre-plan',
        label: 'Pre-plan — no tfplan',
        content: JSON.stringify({
            schemaVersion: 1,
            stage: 'pre_plan',
            run: {
                id: 'run-1',
                createdBy: 'sa/my-group/deployer',
                isDestroy: false,
                refresh: true,
                refreshOnly: false,
                speculative: true,
                terraformVersion: '1.9.8',
                targetAddresses: ['aws_s3_bucket.data'],
                moduleSource: null,
                moduleVersion: null,
                moduleDigest: null,
                configurationVersionId: 'cv-1',
                workspacePath: 'my-group/my-workspace',
                workspaceId: 'ws-1',
                variables: [
                    { key: 'environment', category: 'terraform', sensitive: false, namespacePath: 'my-group', value: 'dev' },
                ],
            },
        }, null, 2),
    },
];
