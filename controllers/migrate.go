package controllers

import (
	"context"

	"github.com/go-logr/logr"
	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

	// Delete job
	if err := r.Delete(context.TODO(), job, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
		return ctrl.Result{}, err
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
	// Update thermo-center instance with annotation. This will trigger a new reconcile cycle.
	if err := r.Status().Update(context.TODO(), i); err != nil {
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
	enableServiceLinks := false
	runAsNonRoot := true
	image := setImageTag(i, "rkojedzinszky/thermo-center-"+r.api.component())

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
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "migrate",
						Image:   image,
						Command: []string{"python", "manage.py", "migrate"},
						EnvFrom: []corev1.EnvFromSource{{
							SecretRef: &v1.SecretEnvSource{LocalObjectReference: v1.LocalObjectReference{Name: thermoCenterSecretName(i)}},
						}},
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("10m"),
								v1.ResourceMemory: resource.MustParse("48Mi"),
							},
						},
					}},
					EnableServiceLinks: &enableServiceLinks,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
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
