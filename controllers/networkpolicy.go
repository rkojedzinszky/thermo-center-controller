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
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update

func (r *ThermoCenterReconciler) reconcileNetworkPolicy(i *kojedzinv1alpha1.ThermoCenter) error {
	policyName := i.Name + "-default"
	policy := &networking.NetworkPolicy{}
	found := true

	err := r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: policyName}, policy)

	if err != nil {
		if errors.IsNotFound(err) != true {
			return err
		}

		found = false

		policy.ObjectMeta = metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      policyName,
		}

		if err = controllerutil.SetControllerReference(i, policy, r.Scheme); err != nil {
			return err
		}
	}

	policy.Spec = networking.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				ThermoCenterInstanceLabel: i.Name,
			}},
		PolicyTypes: []networking.PolicyType{
			networking.PolicyTypeIngress,
		},
		Ingress: []networking.NetworkPolicyIngressRule{
			{
				From: []networking.NetworkPolicyPeer{
					{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								ThermoCenterInstanceLabel: i.Name,
							},
						},
					},
				},
			},
		},
	}

	if found {
		err = r.Update(context.TODO(), policy)
	} else {
		err = r.Create(context.TODO(), policy)
	}

	if err != nil {
		return err
	}

	policyName = i.Name + "-ingress"
	policy = &networking.NetworkPolicy{}
	found = true

	err = r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: policyName}, policy)

	if err != nil {
		if errors.IsNotFound(err) != true {
			return err
		}

		found = false

		policy.ObjectMeta = metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      policyName,
		}

		if err = controllerutil.SetControllerReference(i, policy, r.Scheme); err != nil {
			return err
		}
	}

	httpPort := intstr.FromInt(8080)

	policy.Spec = networking.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				ThermoCenterInstanceLabel: i.Name,
			}},
		PolicyTypes: []networking.PolicyType{
			networking.PolicyTypeIngress,
		},
		Ingress: []networking.NetworkPolicyIngressRule{
			{
				Ports: []networking.NetworkPolicyPort{
					{
						Port: &httpPort,
					},
				},
				From: []networking.NetworkPolicyPeer{
					{
						IPBlock: &networking.IPBlock{
							CIDR: "0.0.0.0/0",
						},
					},
				},
			},
		},
	}

	if found {
		return r.Update(context.TODO(), policy)
	}

	return r.Create(context.TODO(), policy)
}
