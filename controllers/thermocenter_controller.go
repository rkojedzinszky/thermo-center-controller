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
	"encoding/base64"
	"math/rand"
	"sync"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kojedzinv1alpha1 "github.com/rkojedzinszky/thermo-center-controller/api/v1alpha1"
)

// ThermoCenterReconciler reconciles a ThermoCenter object
type ThermoCenterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	rand     *rand.Rand
	randLock *sync.Mutex

	memcached *memcachedReconciler
	mqtt      *mqttReconciler
	ui        *uiReconciler
	grpc      *grpcReconciler
	receiver  *receiverReconciler
	api       *apiReconciler
	ws        *wsReconciler
}

// NewThermoCenterReconciler instantiates a new ThermoCenter Reconciler
func NewThermoCenterReconciler(mgr manager.Manager) *ThermoCenterReconciler {
	return &ThermoCenterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ThermoCenter"),
		Scheme: mgr.GetScheme(),

		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		randLock: &sync.Mutex{},

		mqtt:      &mqttReconciler{},
		memcached: &memcachedReconciler{},
		ui:        &uiReconciler{},
		grpc:      &grpcReconciler{},
		receiver:  &receiverReconciler{},
		api:       &apiReconciler{},
		ws:        &wsReconciler{},
	}
}

// +kubebuilder:rbac:groups=kojedz.in,resources=thermocenters,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=kojedz.in,resources=thermocenters/status,verbs=get;update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;create;update

func (r *ThermoCenterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("thermocenter", req.NamespacedName)

	// Fetch the ThermoCenter instance
	instance := &kojedzinv1alpha1.ThermoCenter{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Reconcile secret
	err = r.reconcileSecret(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile network policies
	err = r.reconcileNetworkPolicy(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Process existing migration Job
	job, err := r.fetchMigrationJob(instance, reqLogger)
	if err != nil {
		return ctrl.Result{}, err
	}
	if job != nil {
		return r.handleMigrationJob(instance, job, reqLogger)
	}

	// Create migration Job if needed
	if r.needsMigration(instance, reqLogger) {
		return r.createMigrationJob(instance, reqLogger)
	}

	// Set state to ready
	instance.Status.Status = "ready"
	if err = r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile ingress
	err = r.reconcileIngress(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile deployments
	for _, rec := range []deploymentReconciler{r.mqtt, r.memcached, r.ui, r.grpc, r.receiver, r.api, r.ws} {
		err = r.reconcile(instance, rec)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ThermoCenterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kojedzinv1alpha1.ThermoCenter{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&source.Kind{Type: &batchv1.Job{}}, &handler.EnqueueRequestForOwner{
			OwnerType: &kojedzinv1alpha1.ThermoCenter{},
		}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(event.CreateEvent) bool {
				return false
			},
		})).
		Complete(r)
}

func (r *ThermoCenterReconciler) getPodSpec(i *kojedzinv1alpha1.ThermoCenter, rec deploymentReconciler) *v1.PodSpec {
	enableServiceLinks := false
	allowPrivilegeEscalation := false
	runAsNonRoot := true

	dep := rec.getDeployment(i)

	// Assign image
	var image string
	if dep != nil && dep.Image != "" {
		image = dep.Image
	} else {
		image = getImagePrefix(i) + rec.component()
	}

	ps := &v1.PodSpec{
		Containers: []v1.Container{{
			Name:  rec.component(),
			Image: setImageTag(i, image),
			SecurityContext: &v1.SecurityContext{
				AllowPrivilegeEscalation: &allowPrivilegeEscalation,
			},
		}},
		EnableServiceLinks: &enableServiceLinks,
		SecurityContext: &v1.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
		},
	}

	if dep != nil {
		ps.Affinity = dep.Affinity
		ps.NodeSelector = dep.NodeSelector
		ps.Tolerations = dep.Tolerations
	}

	return rec.customizePodSpec(r, i, ps)
}

// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update

func (r *ThermoCenterReconciler) reconcile(i *kojedzinv1alpha1.ThermoCenter, rec deploymentReconciler) error {
	var err error

	//
	// Create deployment
	//
	deploymentName := thermoCenterDeploymentName(i, rec)
	deployment := &appsv1.Deployment{}
	deploymentExists := true

	err = r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: deploymentName}, deployment)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		deploymentExists = false

		ls := labelsForComponent(i, rec.component())

		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: i.Namespace,
				Name:      deploymentName,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: ls,
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: ls,
					},
				},
			},
		}

		if err = controllerutil.SetControllerReference(i, deployment, r.Scheme); err != nil {
			return err
		}
	}

	// Set podSpec
	ps := r.getPodSpec(i, rec)

	if ps != nil {
		deployment.Spec.Template.Spec = *ps
		dep := rec.getDeployment(i)
		// Assign replicas
		if dep != nil && dep.Replicas != nil {
			deployment.Spec.Replicas = dep.Replicas
		} else {
			deployment.Spec.Replicas = &i.Spec.Replicas
		}

		// Final deployment customization
		rec.customizeDeployment(r, i, deployment)

		if deploymentExists {
			err = r.Update(context.TODO(), deployment)
		} else {
			err = r.Create(context.TODO(), deployment)
		}
	} else if deploymentExists {
		err = r.Delete(context.TODO(), deployment)
	}

	if err != nil {
		return err
	}

	//
	// Reconcile service
	serviceName := thermoCenterServiceName(i, rec)
	service := &v1.Service{}
	serviceExists := true

	err = r.Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: serviceName}, service)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		serviceExists = false

		if deployment != nil {
			service = &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: i.Namespace,
					Name:      serviceName,
				},
				Spec: v1.ServiceSpec{
					Selector: deployment.Spec.Template.Labels,
				},
			}

			if err = controllerutil.SetControllerReference(deployment, service, r.Scheme); err != nil {
				return err
			}
		}
	}

	origService := service
	if deployment != nil {
		service = rec.customizeService(r, i, service)
	} else {
		service = nil
	}

	if service != nil {
		if serviceExists {
			err = r.Update(context.TODO(), service)
		} else {
			err = r.Create(context.TODO(), service)
		}
	} else if serviceExists {
		err = r.Delete(context.TODO(), origService)
	}

	return err
}

// deploymentReconciler is responsible for exactly one deployment and one service only
type deploymentReconciler interface {
	component() string

	// Extract kojedzinv1alpha1.Deployment to use to construct appsv1.Deployment
	getDeployment(*kojedzinv1alpha1.ThermoCenter) *kojedzinv1alpha1.Deployment

	// Customize v1.PodSpec
	customizePodSpec(*ThermoCenterReconciler, *kojedzinv1alpha1.ThermoCenter, *v1.PodSpec) *v1.PodSpec

	// Customize corev1.Service
	customizeService(*ThermoCenterReconciler, *kojedzinv1alpha1.ThermoCenter, *v1.Service) *v1.Service

	// Customize appsv1.Deployment
	customizeDeployment(*ThermoCenterReconciler, *kojedzinv1alpha1.ThermoCenter, *appsv1.Deployment)
}

type defaultDeploymentReconciler struct{}

//lint:ignore U1000 noop function to comply with interface
// Customize v1.PodSpec
func (*defaultDeploymentReconciler) customizePodSpec(_ *ThermoCenterReconciler, _ *kojedzinv1alpha1.ThermoCenter, ps *v1.PodSpec) *v1.PodSpec {
	return ps
}

//lint:ignore U1000 noop function to comply with interface
// Customize corev1.Service
func (*defaultDeploymentReconciler) customizeService(_ *ThermoCenterReconciler, _ *kojedzinv1alpha1.ThermoCenter, svc *v1.Service) *v1.Service {
	return svc
}

//lint:ignore U1000 noop function to comply with interface
// Customize appsv1.Deployment
func (*defaultDeploymentReconciler) customizeDeployment(_ *ThermoCenterReconciler, _ *kojedzinv1alpha1.ThermoCenter, _ *appsv1.Deployment) {
}

func thermoCenterDeploymentName(i *kojedzinv1alpha1.ThermoCenter, rec deploymentReconciler) string {
	return i.Name + "-" + rec.component()
}

func thermoCenterServiceName(i *kojedzinv1alpha1.ThermoCenter, rec deploymentReconciler) string {
	return i.Name + "-" + rec.component()
}

func thermoCenterSecretName(i *kojedzinv1alpha1.ThermoCenter) string {
	return i.Name + "-secret"
}

func thermoCenterIngressName(i *kojedzinv1alpha1.ThermoCenter) string {
	return i.Name + "-ingress"
}

func thermoCenterMigrationJobName(i *kojedzinv1alpha1.ThermoCenter) string {
	return i.Name + "-migrate"
}

func (r *ThermoCenterReconciler) randomString(len int) string {
	b := r.randomBytes(len * 3 / 4)

	return base64.StdEncoding.EncodeToString(b)
}

func (r *ThermoCenterReconciler) randomBytes(len int) []byte {
	p := make([]byte, len)

	r.randLock.Lock()
	defer r.randLock.Unlock()

	r.rand.Read(p)

	return p
}
