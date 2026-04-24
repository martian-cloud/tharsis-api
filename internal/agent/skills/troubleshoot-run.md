---
name: troubleshoot-run
description: Diagnose failed or problematic Tharsis runs with deep error analysis, log inspection, and actionable remediation steps. Use when asked to diagnose, debug, troubleshoot, or fix a failed run
---

## Run Diagnostics

When asked to diagnose a run, perform deep analysis beyond just reading logs. Gather context, identify root causes, and suggest specific fixes.

## Step 1: Identify the Run

If the user provides a run ID, use it directly.

## Step 2: Gather Full Context

Collect all relevant information in parallel:

1. **Run details**: Use `get_run` to get status, error messages, plan/apply IDs, whether it's a destroy run, configuration version ID, and module source.

2. **Apply Status**: If the run reached the apply stage (status is `errored` and apply ID exists), check the `error_message` field to get the error.

2. **Plan Status**: Use `get_plan` with the plan ID from the run to get the plan details, if the plan failed check the `error_message` field to get the error.

3. **Job Logs**: Only if the error message is not clear, use `get_job_logs` with that latest job ID to get additional information on why the plan/apply failed.

## Step 3: Analyze the Error

Parse the logs and error messages against these known patterns:

### Authentication & Authorization Errors
- **"401 Unauthorized"** or **"403 Forbidden"** → Managed identity issue. The workspace may not have a managed identity assigned, or the identity doesn't have sufficient permissions for the target cloud provider. Suggest: check managed identity assignment, verify IAM role permissions.
- **"no managed identity assigned"** → Workspace needs a managed identity. Suggest: `assign_managed_identity`.
- **"failed to get credentials"** or **"NoCredentialProviders"** → AWS credential chain failure. The managed identity's trust policy may not include the Tharsis runner, or the role ARN is wrong.
- **"token has expired"** or **"ExpiredToken"** → Temporary credentials expired during a long-running operation. Suggest: increase max_job_duration or optimize the configuration.

### Provider & Registry Errors
- **"failed to query available provider packages"** or **"registry.terraform.io"** → Provider download failure. Could be network issue or private registry auth. Check if the module uses private providers from the Tharsis registry.
- **"failed to install provider"** → Provider version constraint issue or registry unreachable.
- **"Module not found"** or **"module source"** → Module source URL is wrong or the module version doesn't exist in the registry.

### State & Lock Errors
- **"state lock"** or **"Lock Info"** → Another run or process holds the state lock. Check if there's a concurrent run on the same workspace. Suggest: wait for the other run to complete, or force-unlock if it's stale.
- **"state snapshot was created by Terraform"** with version mismatch → Terraform version in workspace doesn't match what created the state. Suggest: update workspace Terraform version.

### Plan Errors
- **"Error: Reference to undeclared"** → Variable or resource reference doesn't exist in the configuration. Likely a typo or missing variable definition.
- **"Error: Missing required variable"** → A variable declared in the config isn't set on the workspace. Suggest: `set_variable` with the missing variable name.
- **"Error: Unsupported attribute"** → Resource attribute doesn't exist for the provider version being used. Could be a provider version mismatch.
- **"Error: Invalid value for variable"** → Variable value doesn't match the type constraint or validation rules.

### Apply Errors
- **"Error: creating"** or **"Error: updating"** → Cloud API error during resource creation/update. Parse the specific AWS/GCP/Azure error message for details.
- **"Error: deleting"** with **"prevent_destroy_plan"** → Workspace has destroy protection enabled. If the destroy is intentional, suggest updating the workspace setting.
- **"ResourceAlreadyExistsException"** or **"AlreadyExists"** → Trying to create a resource that already exists. May need to import it into state first.
- **"DependencyViolation"** → Trying to delete a resource that other resources depend on. Check resource ordering.

### Configuration Errors
- **"configuration version"** errors → The uploaded configuration may be invalid. Check if the upload completed successfully.
- **"No configuration provided"** → Run was created without a configuration version or module source.

## Step 4: Present Diagnosis

Structure the response as:

```
## Run Diagnosis

**Run**: <run_id>
**Workspace**: <workspace_path>
**Status**: <status>
**Stage**: <plan or apply>
**Terraform Version**: <version>

### Error
<Clear, concise description of what went wrong>

### Root Cause
<Why it happened — the underlying issue, not just the symptom>

### Evidence
<Relevant log excerpts — keep brief, highlight the key lines>

### Recommended Fix
<Specific, actionable steps.>
1. <step>
2. <step>
```
