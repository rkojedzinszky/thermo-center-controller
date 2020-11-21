/*
MIT License

Copyright (c) 2020 Richard Kojedzinszky

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controllers

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
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

func getImagePrefix(i *kojedzinv1alpha1.ThermoCenter) string {
	if i.Spec.Version != nil && semver.Compare(fmt.Sprintf("v%s", *i.Spec.Version), "v4") < 0 {
		return "docker.io/rkojedzinszky/thermo-center-"
	}

	return "ghcr.io/rkojedzinszky/thermo-center-"
}
