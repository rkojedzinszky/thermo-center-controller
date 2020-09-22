package controllers

import (
	"context"
	"fmt"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:resources=secrets,verbs=get;create;update;patch

const (
	sDBHOST       = "DBHOST"
	sDBPORT       = "DBPORT"
	sDBNAME       = "DBNAME"
	sDBUSER       = "DBUSER"
	sDBPASSWORD   = "DBPASSWORD"
	sDBCONNMAXAGE = "DBCONNMAXAGE"
	sSECRETKEY    = "SECRET_KEY"
)

// Reconcile secret for thermo-center
func (r *ThermoCenterReconciler) reconcileSecret(i *kojedzinv1alpha1.ThermoCenter) error {
	secretName := thermoCenterSecretName(i)
	secret := &corev1.Secret{}
	found := true

	err := r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: secretName}, secret)
	if err != nil {
		if errors.IsNotFound(err) != true {
			return err
		}

		found = false

		// Create object
		secret.ObjectMeta = metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      secretName,
		}

		if err = controllerutil.SetControllerReference(i, secret, r.Scheme); err != nil {
			return err
		}

		secret.Data = make(map[string][]byte)
		secret.Data[sDBCONNMAXAGE] = []byte("0")
		secret.Data[sSECRETKEY] = []byte(r.randomString(64))
	}

	// Overwrite fields
	secret.Data[sDBHOST] = []byte(i.Spec.Database.Host)
	secret.Data[sDBPORT] = []byte(fmt.Sprintf("%d", i.Spec.Database.Port))
	secret.Data[sDBNAME] = []byte(i.Spec.Database.Name)
	secret.Data[sDBUSER] = []byte(i.Spec.Database.User)
	secret.Data[sDBPASSWORD] = []byte(i.Spec.Database.Password)

	if found {
		return r.Update(context.TODO(), secret)
	}

	return r.Create(context.TODO(), secret)
}
