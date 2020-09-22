package controllers

import (
	"sort"
	"strings"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	// ThermoCenterInstanceLabel represents common label for a ThermoCenter instance
	ThermoCenterInstanceLabel = "thermo-center-instance"
)

// Generate labels for specific components
func labelsForComponent(instance *kojedzinv1alpha1.ThermoCenter, component string) map[string]string {
	return map[string]string{
		ThermoCenterInstanceLabel: instance.Name,
		"thermo-center-component": component,
	}
}

func replicas(r int32) *int32 {
	return &r
}

// Generate image name based on desired version
func setImageTag(i *kojedzinv1alpha1.ThermoCenter, image string) string {
	if strings.IndexByte(image, ':') != -1 {
		return image
	}

	tag := "latest"

	if i.Spec.Version != nil && *i.Spec.Version != "" {
		tag = *i.Spec.Version
	}

	return image + ":" + tag
}

// Merge all variables from src into dst
func mergeEnvironmentVariables(dst *[]v1.EnvVar, src map[string]string) {
	// Iterate over existing keys
	for idx := range *dst {
		key := (*dst)[idx].Name
		if srcvalue, ok := src[key]; ok {
			(*dst)[idx].Value = srcvalue
			(*dst)[idx].ValueFrom = nil

			delete(src, key)
		}
	}

	// Add remaining keys in deterministic order
	var keys []string
	for key := range src {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		*dst = append(*dst, v1.EnvVar{
			Name:  key,
			Value: src[key],
		})
	}
}
