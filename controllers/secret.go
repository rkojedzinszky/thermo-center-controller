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
	"fmt"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update

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
	secret := &v1.Secret{}
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
