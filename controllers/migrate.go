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

	"github.com/go-logr/logr"
	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;delete

const thermoCenterDBVersionAnnotation = "thermo-center-db-version"

func (r *ThermoCenterReconciler) needsMigration(i *kojedzinv1alpha1.ThermoCenter, l logr.Logger) bool {
	// latest image not handled by operator
	if i.Spec.Version == nil || *i.Spec.Version == "" {
		l.Info("No version specified, not doing migration")

		return false
	}

	// No migration needed if annotation matches desired version
	if i.Status.DatabaseVersion == *i.Spec.Version {
		l.Info("Desired and current databae versions match, skipping migration")

		return false
	}

	return true
}

func (r *ThermoCenterReconciler) fetchMigrationJob(i *kojedzinv1alpha1.ThermoCenter, l logr.Logger) (*batchv1.Job, error) {
	jobName := thermoCenterMigrationJobName(i)
	job := &batchv1.Job{}

	l = l.WithValues("jobName", jobName)

	err := r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: jobName}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return job, nil
}

func (r *ThermoCenterReconciler) handleMigrationJob(i *kojedzinv1alpha1.ThermoCenter, job *batchv1.Job, l logr.Logger) (ctrl.Result, error) {
	// Wait for completion
	if len(job.Status.Conditions) == 0 {
		return ctrl.Result{}, nil
	}

	// Requeue the job if failed
	if job.Status.Conditions[0].Type == batchv1.JobFailed {
		l.Info("Migration job failed, requeueing")

		i.Status.Status = "migration failed"
	} else {
		l.Info("Migration job succeeded")

		// Update db version from job to annotation
		i.Status.Status = "migration done"
		i.Status.DatabaseVersion = job.Annotations[thermoCenterDBVersionAnnotation]
	}

	l.Info("Updating Status")
	// Update thermo-center instance status.
	if err := r.Status().Update(context.TODO(), i); err != nil {
		return ctrl.Result{}, err
	}

	// Delete job. This will trigger new reconcile cycle.
	if err := r.Delete(context.TODO(), job, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ThermoCenterReconciler) createMigrationJob(i *kojedzinv1alpha1.ThermoCenter, l logr.Logger) (ctrl.Result, error) {
	i.Status.Status = "migrating"

	l.Info("Updating Status")
	// Update thermo-center instance with annotation. This will trigger a new reconcile cycle.
	if err := r.Status().Update(context.TODO(), i); err != nil {
		return ctrl.Result{}, err
	}

	// Create migration job
	l.Info("Creating migration job", "targetVersion", *i.Spec.Version)

	var activeDeadlineSeconds int64 = 600

	// Customize api POD for migration
	ps := r.getPodSpec(i, r.api)
	ps.Containers[0].Name = "migrate"
	ps.Containers[0].Command = []string{"python", "manage.py", "migrate"}
	ps.Containers[0].LivenessProbe = nil
	ps.Containers[0].ReadinessProbe = nil
	ps.RestartPolicy = v1.RestartPolicyNever

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      thermoCenterMigrationJobName(i),
			Annotations: map[string]string{
				thermoCenterDBVersionAnnotation: *i.Spec.Version,
			},
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &activeDeadlineSeconds,
			Template: v1.PodTemplateSpec{
				Spec: *ps,
			},
		},
	}

	if err := controllerutil.SetControllerReference(i, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Create(context.TODO(), job); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
