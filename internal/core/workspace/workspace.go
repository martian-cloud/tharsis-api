package workspace

// StateVersionOutputLimitViolationMsg is the stable marker embedded in the error message returned by
// WorkspaceService.CreateStateVersion when the state version itself was created successfully
// but one or more outputs were rejected for exceeding the per-output size limit or the
// ResourceLimitOutputsPerStateVersion count limit. It is a wire-contract between the API server
// and the job executor: the executor matches on this substring (in the gRPC status message) to
// tell this specific partial-success case apart from other InvalidArgument/state-version-creation
// failures.
const StateVersionOutputLimitViolationMsg = "state version output limit exceeded"
