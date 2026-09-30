package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	dispatchertypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// testJobUUID is a realistic v7 UUID; testJobGID is its global ID as DispatchJob receives it.
const testJobUUID = "01a0c99e-9fb2-7b3d-8093-9cc911484370"

func testJobGID(t *testing.T) string {
	t.Helper()
	return gid.ToGlobalID(types.JobModelType, testJobUUID)
}

func baseInClusterData() map[string]string {
	return map[string]string{
		"endpoint":       "https://api.example.com",
		"image":          "test-image:latest",
		"memory_request": "128Mi",
		"memory_limit":   "256Mi",
		"auth_type":      AuthTypeInCluster,
	}
}

func envValue(env []corev1.EnvVar, name string) (string, bool) {
	for _, e := range env {
		if e.Name == name {
			return e.Value, true
		}
	}

	return "", false
}

func Test_parseConfig_errors(t *testing.T) {
	withData := func(extra map[string]string) map[string]string {
		data := baseInClusterData()
		for k, v := range extra {
			data[k] = v
		}

		return data
	}

	tests := []struct {
		pluginData          map[string]string
		name                string
		expectedErrContains string
		wantErr             bool
	}{
		{
			name:                "missing required field",
			pluginData:          map[string]string{"endpoint": "https://api.example.com"},
			wantErr:             true,
			expectedErrContains: "kubernetes job dispatcher requires plugin data",
		},
		{
			name:                "empty memory_limit is rejected",
			pluginData:          withData(map[string]string{"memory_limit": ""}),
			wantErr:             true,
			expectedErrContains: "memory_limit must be a non-zero value",
		},
		{
			name:                "invalid security context",
			pluginData:          withData(map[string]string{"security_context_run_as_user": "not-a-number"}),
			wantErr:             true,
			expectedErrContains: "failed to parse security_context_run_as_user",
		},
		{
			name:       "empty security context value is skipped",
			pluginData: withData(map[string]string{"security_context_run_as_user": ""}),
			wantErr:    false,
		},
		{
			name:       "empty cpu_request is skipped",
			pluginData: withData(map[string]string{"cpu_request": ""}),
			wantErr:    false,
		},
		{
			name:                "invalid node selector",
			pluginData:          withData(map[string]string{"node_selector": "kubernetes.io/arch"}),
			wantErr:             true,
			expectedErrContains: "invalid node selector format",
		},
		{
			name:                "invalid pod_active_deadline_seconds",
			pluginData:          withData(map[string]string{"pod_active_deadline_seconds": "not-a-number"}),
			wantErr:             true,
			expectedErrContains: "failed to parse pod_active_deadline_seconds",
		},
		{
			name:                "out-of-range pod_active_deadline_seconds",
			pluginData:          withData(map[string]string{"pod_active_deadline_seconds": "0"}),
			wantErr:             true,
			expectedErrContains: "pod_active_deadline_seconds must be a positive value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger, _ := logger.NewForTest()
			cfg, err := parseConfig(tt.pluginData, "discovery.example.com", testLogger)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	}
}

func Test_podName(t *testing.T) {
	name, err := podName(testJobGID(t))
	require.NoError(t, err)
	assert.Equal(t, "tharsis-job-"+testJobUUID, name)
	assert.LessOrEqual(t, len(name), 63)

	_, err = podName("not-a-global-id")
	assert.Error(t, err)
}

func TestConfig_buildPod_ownerLabelWinsOverUserLabels(t *testing.T) {
	jobID := testJobGID(t)
	c := &config{
		apiEndpoint: "http://localhost",
		limits:      &dispatchertypes.ResourceLimits{},
		labels: map[string]string{
			ownerLabelKey: "attacker-value",
			jobIDLabelKey: "spoofed",
			"team":        "platform",
		},
	}

	pod, err := c.buildPod(jobID, "tok")
	require.NoError(t, err)

	assert.Equal(t, ownerLabelValue, pod.Labels[ownerLabelKey])
	assert.Equal(t, jobID, pod.Labels[jobIDLabelKey])
	assert.Equal(t, "platform", pod.Labels["team"])
}

func TestConfig_buildPod_nameFromDecodedUUID(t *testing.T) {
	c := &config{limits: &dispatchertypes.ResourceLimits{}}
	pod, err := c.buildPod(testJobGID(t), "tok")
	require.NoError(t, err)
	assert.Equal(t, "tharsis-job-"+testJobUUID, pod.Name)
	assert.Empty(t, pod.GenerateName)
	assert.LessOrEqual(t, len(pod.Name), 63)
}

func TestConfig_buildPod_invalidJobID(t *testing.T) {
	c := &config{limits: &dispatchertypes.ResourceLimits{}}
	_, err := c.buildPod("not-a-global-id", "tok")
	assert.Error(t, err)
}

func TestConfig_buildPod_limitEnvVars(t *testing.T) {
	c := &config{
		apiEndpoint: "http://localhost",
		limits: &dispatchertypes.ResourceLimits{
			MemoryBytes:   256,
			DiskReadBytes: 512,
		},
	}

	pod, err := c.buildPod(testJobGID(t), "tok")
	require.NoError(t, err)
	require.Len(t, pod.Spec.Containers, 1)
	env := pod.Spec.Containers[0].Env

	mem, ok := envValue(env, "MEMORY_LIMIT")
	assert.True(t, ok)
	assert.Equal(t, "256", mem)

	disk, ok := envValue(env, "DISK_READ_LIMIT")
	assert.True(t, ok)
	assert.Equal(t, "512", disk)

	jobID, ok := envValue(env, "JOB_ID")
	assert.True(t, ok)
	assert.Equal(t, testJobGID(t), jobID)
}

func TestConfig_buildPod_disablesServiceLinks(t *testing.T) {
	c := &config{limits: &dispatchertypes.ResourceLimits{}}
	pod, err := c.buildPod(testJobGID(t), "tok")
	require.NoError(t, err)
	require.NotNil(t, pod.Spec.EnableServiceLinks)
	assert.False(t, *pod.Spec.EnableServiceLinks)
	require.NotNil(t, pod.Spec.AutomountServiceAccountToken)
	assert.False(t, *pod.Spec.AutomountServiceAccountToken)
}
