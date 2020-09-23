package controllers

import (
	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type receiverReconciler struct {
}

func (rec *receiverReconciler) component() string {
	return "receiver"
}

func (rec *receiverReconciler) getDeployment(i *kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment {
	return i.Spec.Receiver
}

func (rec *receiverReconciler) customizeDeployment(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, deployment *appsv1.Deployment) *appsv1.Deployment {
	// Resource requirements
	deployment.Spec.Template.Spec.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("16Mi"),
		},
		Limits: v1.ResourceList{
			"hardware/cc1101": resource.MustParse("1"),
		},
	}

	deployment.Spec.Template.Spec.Containers[0].Env = []v1.EnvVar{
		{
			Name:  "GRPCSERVER_HOST",
			Value: thermoCenterServiceName(i, r.grpc),
		},
	}

	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{IntVal: 1},
		},
	}

	return deployment
}

func (rec *receiverReconciler) customizeService(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, service *v1.Service) *v1.Service {
	service.Spec.Ports = []v1.ServicePort{{
		Name: "grpc",
		Port: 8079,
	}}

	return service
}
