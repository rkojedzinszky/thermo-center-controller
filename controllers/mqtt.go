package controllers

import (
	"strconv"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var mqttDeployment = &kojedzinv1alpha1.Deployment{
	Image:    "eclipse-mosquitto:1.6.12",
	Replicas: replicas(1),
}

type mqttReconciler struct {
}

func (m *mqttReconciler) component() string {
	return "mqtt"
}

func (m *mqttReconciler) getDeployment(i *kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment {
	return mqttDeployment
}

func (m *mqttReconciler) customizeDeployment(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, deployment *appsv1.Deployment) *appsv1.Deployment {
	if i.Spec.ExternalMQTT != nil {
		return nil
	}

	// Resource requirements
	deployment.Spec.Template.Spec.Containers[0].Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("10m"),
			v1.ResourceMemory: resource.MustParse("8Mi"),
		},
	}

	// Override uid/gid
	runAsUser := int64(1883)
	runAsGroup := int64(1883)

	deployment.Spec.Template.Spec.SecurityContext.RunAsUser = &runAsUser
	deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = &runAsGroup

	// Ports
	deployment.Spec.Template.Spec.Containers[0].Ports = []v1.ContainerPort{{
		Name:          m.component(),
		ContainerPort: 1883,
	}}

	return deployment
}

func (m *mqttReconciler) customizeService(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, service *v1.Service) *v1.Service {
	service.Spec.Ports = []v1.ServicePort{{
		Name: m.component(),
		Port: 1883,
	}}

	return service
}

func (m *mqttReconciler) serviceHost(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter) string {
	if i.Spec.ExternalMQTT == nil {
		return thermoCenterServiceName(i, m)
	}

	return i.Spec.ExternalMQTT.Hostname
}

func (m *mqttReconciler) servicePort(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter) int {
	if i.Spec.ExternalMQTT == nil {
		return 1883
	}

	port := i.Spec.ExternalMQTT.Port
	if port == 0 {
		port = 1883
	}

	return port
}

func (m *mqttReconciler) setEnvironment(r *ThermoCenterReconciler, i *kojedzinv1alpha1.ThermoCenter, env *[]v1.EnvVar) {
	mergeEnvironmentVariables(env, map[string]string{
		"MQTT_HOST": m.serviceHost(r, i),
		"MQTT_PORT": strconv.Itoa(m.servicePort(r, i)),
	})
}
