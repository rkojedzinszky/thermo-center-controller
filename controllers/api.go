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
	"strings"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type apiReconciler struct {
}

func (api *apiReconciler) component() string {
	return "api"
}

func (api *apiReconciler) getDeployment(i *kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment {
	return i.Spec.API
}

func (api *apiReconciler) customizePodSpec(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, ps *v1.PodSpec) *v1.PodSpec {
	// Resource requirements
	ps.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("48Mi"),
		},
	}

	ps.Containers[0].EnvFrom = append(ps.Containers[0].EnvFrom,
		v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: thermoCenterSecretName(i)}},
		},
	)

	allowedHosts := strings.Join(i.Spec.Ingress.HostNames, ",")
	if allowedHosts == "" {
		allowedHosts = "undefined.domain.name"
	}

	ps.Containers[0].Env = append(ps.Containers[0].Env,
		v1.EnvVar{
			Name:  "ALLOWED_HOSTS",
			Value: allowedHosts,
		},
		v1.EnvVar{
			Name:  "RECEIVER_HOST",
			Value: thermoCenterServiceName(i, r.receiver),
		},
	)

	r.memcached.setEnvironment(r, i, &ps.Containers[0].Env)

	for _, host := range i.Spec.Ingress.HostNames {
		if host != "" {
			ps.Containers[0].ReadinessProbe = &v1.Probe{
				Handler: v1.Handler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.IntOrString{IntVal: 8080},
						HTTPHeaders: []v1.HTTPHeader{
							{
								Name:  "Host",
								Value: host,
							},
						},
					},
				},
			}
			break
		}
	}

	return ps
}

func (api *apiReconciler) customizeService(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, service *v1.Service) *v1.Service {
	service.Spec.Ports = []v1.ServicePort{{
		Name: "http",
		Port: 8080,
	}}

	return service
}
