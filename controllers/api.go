package controllers

import (
	"strings"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
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

func (api *apiReconciler) customizeDeployment(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, deployment *appsv1.Deployment) *appsv1.Deployment {
	// Resource requirements
	deployment.Spec.Template.Spec.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("48Mi"),
		},
	}

	deployment.Spec.Template.Spec.Containers[0].EnvFrom = []v1.EnvFromSource{
		{
			SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: thermoCenterSecretName(i)}},
		},
	}

	allowedHosts := strings.Join(i.Spec.Ingress.HostNames, ",")
	if allowedHosts == "" {
		allowedHosts = "undefined.domain.name"
	}

	deployment.Spec.Template.Spec.Containers[0].Env = []v1.EnvVar{
		{
			Name:  "ALLOWED_HOSTS",
			Value: allowedHosts,
		},
		{
			Name:  "RECEIVER_HOST",
			Value: r.serviceName(i, r.receiver),
		},
	}

	r.memcached.setEnvironment(r, i, &deployment.Spec.Template.Spec.Containers[0].Env)

	for _, host := range i.Spec.Ingress.HostNames {
		if host != "" {
			deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &v1.Probe{
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

	return deployment
}

func (api *apiReconciler) customizeService(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, service *v1.Service) *v1.Service {
	service.Spec.Ports = []v1.ServicePort{{
		Name: "http",
		Port: 8080,
	}}

	return service
}