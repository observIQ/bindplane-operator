package controller

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

const (
	goMaxProcsEnvVar   = "GOMAXPROCS"
	goMemLimitEnvVar   = "GOMEMLIMIT"
	goRuntimeHeadroomP = int64(10)
)

// getGoRuntimeEnvVars returns resource-aware Go runtime env vars based on the
// effective merged container resources.
func getGoRuntimeEnvVars(resources corev1.ResourceRequirements) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	if maxProcs, ok := getGoMaxProcsValue(resources); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  goMaxProcsEnvVar,
			Value: strconv.FormatInt(maxProcs, 10),
		})
	}

	if memLimit, ok := getGoMemLimitValue(resources); ok {
		envVars = append(envVars, corev1.EnvVar{
			Name:  goMemLimitEnvVar,
			Value: strconv.FormatInt(memLimit, 10),
		})
	}

	return envVars
}

// applyGoRuntimeEnvVars upserts computed Go runtime env vars onto the container.
func applyGoRuntimeEnvVars(container *corev1.Container) {
	if container == nil {
		return
	}
	for _, envVar := range getGoRuntimeEnvVars(container.Resources) {
		upsertEnvVar(&container.Env, envVar)
	}
}

func getGoMaxProcsValue(resources corev1.ResourceRequirements) (int64, bool) {
	if cpuLimit, ok := resources.Limits[corev1.ResourceCPU]; ok && !cpuLimit.IsZero() {
		return ceilMilliCPUs(cpuLimit.MilliValue()), true
	}
	if cpuRequest, ok := resources.Requests[corev1.ResourceCPU]; ok && !cpuRequest.IsZero() {
		return ceilMilliCPUs(cpuRequest.MilliValue()), true
	}
	return 0, false
}

func getGoMemLimitValue(resources corev1.ResourceRequirements) (int64, bool) {
	if memLimit, ok := resources.Limits[corev1.ResourceMemory]; ok && !memLimit.IsZero() {
		return applyMemoryHeadroom(memLimit.Value()), true
	}
	if memRequest, ok := resources.Requests[corev1.ResourceMemory]; ok && !memRequest.IsZero() {
		return applyMemoryHeadroom(memRequest.Value()), true
	}
	return 0, false
}

func ceilMilliCPUs(milliCPU int64) int64 {
	if milliCPU <= 0 {
		return 1
	}
	return (milliCPU + 999) / 1000
}

func applyMemoryHeadroom(bytes int64) int64 {
	if bytes <= 0 {
		return 1
	}
	headroomBytes := bytes * goRuntimeHeadroomP / 100
	if bytes-headroomBytes <= 0 {
		return 1
	}
	return bytes - headroomBytes
}

func upsertEnvVar(envVars *[]corev1.EnvVar, envVar corev1.EnvVar) {
	for i := range *envVars {
		if (*envVars)[i].Name == envVar.Name {
			(*envVars)[i] = envVar
			return
		}
	}
	*envVars = append(*envVars, envVar)
}
