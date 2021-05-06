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
	"context"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update

func (r *ThermoCenterReconciler) reconcileIngressTLSSecretName(i *kojedzinv1alpha1.ThermoCenter) string {
	return i.Name + "-tls-secret"
}

func (r *ThermoCenterReconciler) reconcileIngressTLSSecret(i *kojedzinv1alpha1.ThermoCenter) error {
	if !i.Spec.Ingress.TLS {
		return nil
	}

	secretName := r.reconcileIngressTLSSecretName(i)
	secret := &v1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: secretName}, secret)

	// Return if found or other than not-found error encountered
	if err == nil || !errors.IsNotFound(err) {
		return err
	}

	secret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      secretName,
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": {},
			"tls.crt": {},
		},
	}

	if err = controllerutil.SetControllerReference(i, secret, r.Scheme); err != nil {
		return err
	}

	return r.Create(context.TODO(), secret)
}

func (r *ThermoCenterReconciler) reconcileIngress(i *kojedzinv1alpha1.ThermoCenter) error {
	err := r.reconcileIngressTLSSecret(i)
	if err != nil {
		return err
	}

	ingressName := thermoCenterIngressName(i)
	ingress := &networking.Ingress{}
	found := true

	err = r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: ingressName}, ingress)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		found = false

		ingress.ObjectMeta = metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      ingressName,
		}

		if err = controllerutil.SetControllerReference(i, ingress, r.Scheme); err != nil {
			return err
		}
	}

	ingress.Spec.Rules = nil
	pathType := networking.PathTypePrefix

	for _, host := range i.Spec.Ingress.HostNames {
		ingress.Spec.Rules = append(ingress.Spec.Rules, networking.IngressRule{
			Host: host,
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path:     "/api/",
							PathType: &pathType,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: thermoCenterServiceName(i, r.api),
									Port: networking.ServiceBackendPort{
										Name: "http",
									},
								},
							},
						},
						{
							Path:     "/admin/",
							PathType: &pathType,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: thermoCenterServiceName(i, r.api),
									Port: networking.ServiceBackendPort{
										Name: "http",
									},
								},
							},
						},
						{
							Path:     "/ws/",
							PathType: &pathType,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: thermoCenterServiceName(i, r.ws),
									Port: networking.ServiceBackendPort{
										Name: "http",
									},
								},
							},
						},
						{
							Path:     "/",
							PathType: &pathType,
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: thermoCenterServiceName(i, r.ui),
									Port: networking.ServiceBackendPort{
										Name: "http",
									},
								},
							},
						},
					},
				},
			},
		})
	}

	if i.Spec.Ingress.TLS {
		ingress.Spec.TLS = []networking.IngressTLS{
			{
				Hosts:      i.Spec.Ingress.HostNames,
				SecretName: r.reconcileIngressTLSSecretName(i),
			},
		}
	} else {
		ingress.Spec.TLS = nil
	}

	// Add extra annotations
	if ingress.Annotations == nil {
		ingress.Annotations = make(map[string]string)
	}

	for key, value := range i.Spec.Ingress.Annotations {
		ingress.Annotations[key] = value
	}

	if found {
		return r.Update(context.TODO(), ingress)
	}

	return r.Create(context.TODO(), ingress)
}
