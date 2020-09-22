package controllers

import (
	"context"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;create;update;patch

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

		policy.ObjectMeta = v1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      policyName,
		}

		if err = controllerutil.SetControllerReference(i, policy, r.Scheme); err != nil {
			return err
		}
	}

	policy.Spec = networking.NetworkPolicySpec{
		PodSelector: v1.LabelSelector{
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
						PodSelector: &v1.LabelSelector{
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

		policy.ObjectMeta = v1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      policyName,
		}

		if err = controllerutil.SetControllerReference(i, policy, r.Scheme); err != nil {
			return err
		}
	}

	httpPort := intstr.FromInt(8080)

	policy.Spec = networking.NetworkPolicySpec{
		PodSelector: v1.LabelSelector{
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
