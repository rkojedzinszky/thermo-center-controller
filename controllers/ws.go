package controllers

import (
	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type wsReconciler struct {
}

func (ws *wsReconciler) component() string {
	return "ws"
}

func (ws *wsReconciler) getDeployment(i *kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment {
	return i.Spec.WS
}

func (ws *wsReconciler) customizePodSpec(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, ps *v1.PodSpec) *v1.PodSpec {
	// Resource requirements
	ps.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1m"),
			v1.ResourceMemory: resource.MustParse("16Mi"),
		},
	}

	ps.Containers[0].Env = append(ps.Containers[0].Env,
		v1.EnvVar{
			Name:  "WS_PORT",
			Value: "8080",
		},
		v1.EnvVar{
			Name:  "THERMO_CENTER_API_HOST",
			Value: thermoCenterServiceName(i, r.api),
		},
	)

	r.mqtt.setEnvironment(r, i, &ps.Containers[0].Env)

	return ps
}

func (ws *wsReconciler) customizeService(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, service *v1.Service) *v1.Service {
	service.Spec.Ports = []v1.ServicePort{{
		Name: "http",
		Port: 8080,
	}}

	return service
}
