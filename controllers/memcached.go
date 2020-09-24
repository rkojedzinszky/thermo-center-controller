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
	"strconv"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var memcachedDeployment = &kojedzinv1alpha1.Deployment{
	Image:    "memcached:1.6.6-alpine",
	Replicas: replicas(1),
}

type memcachedReconciler struct {
	defaultDeploymentReconciler
}

func (m *memcachedReconciler) component() string {
	return "memcached"
}

func (m *memcachedReconciler) getDeployment(i *kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment {
	return memcachedDeployment
}

func (m *memcachedReconciler) customizePodSpec(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, ps *v1.PodSpec) *v1.PodSpec {
	if i.Spec.ExternalMemcached != nil {
		return nil
	}

	// Resource requirements
	ps.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("16Mi"),
		},
	}

	// Override uid/gid
	runAsUser := int64(11211)
	runAsGroup := int64(11211)

	ps.SecurityContext.RunAsUser = &runAsUser
	ps.SecurityContext.RunAsGroup = &runAsGroup

	// Command
	ps.Containers[0].Command = []string{"memcached", "-m", "16"}

	// Ports
	ps.Containers[0].Ports = []v1.ContainerPort{{
		Name:          m.component(),
		ContainerPort: 11211,
	}}

	return ps
}

func (m *memcachedReconciler) customizeService(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, service *v1.Service) *v1.Service {
	service.Spec.Ports = []v1.ServicePort{{
		Name: m.component(),
		Port: 11211,
	}}

	return service
}

func (m *memcachedReconciler) serviceHost(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter) string {
	if i.Spec.ExternalMemcached == nil {
		return thermoCenterServiceName(i, m)
	}

	return i.Spec.ExternalMemcached.Hostname
}

func (m *memcachedReconciler) servicePort(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter) int {
	if i.Spec.ExternalMemcached == nil {
		return 11211
	}

	port := i.Spec.ExternalMemcached.Port
	if port == 0 {
		port = 11211
	}

	return port
}

func (m *memcachedReconciler) setEnvironment(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, env *[]v1.EnvVar) {
	*env = append(*env,
		v1.EnvVar{
			Name:  "MEMCACHED_HOST",
			Value: m.serviceHost(r, i),
		},
		v1.EnvVar{
			Name:  "MEMCACHED_PORT",
			Value: strconv.Itoa(m.servicePort(r, i)),
		},
	)
}
