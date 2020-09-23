package controllers

import (
	"strconv"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var memcachedDeployment = &kojedzinv1alpha1.Deployment{
	Image:    "memcached:1.6.6-alpine",
	Replicas: replicas(1),
}

type memcachedReconciler struct {
}

func (m *memcachedReconciler) component() string {
	return "memcached"
}

func (m *memcachedReconciler) getDeployment(i *kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment {
	return memcachedDeployment
}

func (m *memcachedReconciler) customizeDeployment(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, deployment *appsv1.Deployment) *appsv1.Deployment {
	if i.Spec.ExternalMemcached != nil {
		return nil
	}

	// Resource requirements
	deployment.Spec.Template.Spec.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("16Mi"),
		},
	}

	// Override uid/gid
	runAsUser := int64(11211)
	runAsGroup := int64(11211)

	deployment.Spec.Template.Spec.SecurityContext.RunAsUser = &runAsUser
	deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = &runAsGroup

	// Command
	deployment.Spec.Template.Spec.Containers[0].Command = []string{"memcached", "-m", "16"}

	// Ports
	deployment.Spec.Template.Spec.Containers[0].Ports = []v1.ContainerPort{{
		Name:          m.component(),
		ContainerPort: 11211,
	}}

	return deployment
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
	mergeEnvironmentVariables(env, map[string]string{
		"MEMCACHED_HOST": m.serviceHost(r, i),
		"MEMCACHED_PORT": strconv.Itoa(m.servicePort(r, i)),
	})
}
