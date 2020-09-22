package controllers

import (
	"fmt"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type grpcReconciler struct {
}

func (grpc *grpcReconciler) component() string {
	return "grpcserver"
}

func (grpc *grpcReconciler) getDeployment(i *kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment {
	return i.Spec.GRPC
}

func (grpc *grpcReconciler) customizeDeployment(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, deployment *appsv1.Deployment) *appsv1.Deployment {
	// Resource requirements
	deployment.Spec.Template.Spec.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("16Mi"),
		},
	}

	deployment.Spec.Template.Spec.Containers[0].EnvFrom = []v1.EnvFromSource{
		{
			SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: thermoCenterSecretName(i)}},
		},
	}

	deployment.Spec.Template.Spec.Containers[0].Env = nil

	if i.Spec.Graphite.Hostname != "" {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "CARBON_LINE_RECEIVER_HOST",
			Value: i.Spec.Graphite.Hostname,
		})
	}

	if i.Spec.Graphite.Port != 0 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "CARBON_LINE_RECEIVER_PORT",
			Value: fmt.Sprintf("%d", i.Spec.Graphite.Port),
		})
	}

	r.memcached.setEnvironment(r, i, &deployment.Spec.Template.Spec.Containers[0].Env)
	r.mqtt.setEnvironment(r, i, &deployment.Spec.Template.Spec.Containers[0].Env)

	return deployment
}

func (grpc *grpcReconciler) customizeService(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, service *v1.Service) *v1.Service {
	service.Spec.Ports = []v1.ServicePort{{
		Name: "grpc",
		Port: 8079,
	}}

	return service
}
