/*
Copyright 2021 Manuel Rüger <manuel@rueg.eu>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"html/template"
	"reflect"

	"github.com/Masterminds/sprig"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sresourcequotaautoscalerv1beta1 "github.com/mrueg/k8s-resourcequota-autoscaler/api/v1beta1"
)

// ManagedResourceQuotaReconciler reconciles a ManagedResourceQuota object
type ManagedResourceQuotaReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8s-resourcequota-autoscaler.m21r.de,resources=managedresourcequota,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s-resourcequota-autoscaler.m21r.de,resources=managedresourcequota/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s-resourcequota-autoscaler.m21r.de,resources=managedresourcequota/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedResourceQuota object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *ManagedResourceQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedresourcequota", req.NamespacedName)
	managedResourceQuota := &k8sresourcequotaautoscalerv1beta1.ManagedResourceQuota{}
	err := r.Get(ctx, req.NamespacedName, managedResourceQuota)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("ManagedResourceQuota resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ManagedResourceQuota")
		return ctrl.Result{}, err
	}

	nodeList := &corev1.NodeList{}
	err = r.List(ctx, nodeList)

	if err != nil {
		log.Error(err, "Failed to list Nodes")
		return ctrl.Result{}, err
	}

	nodeCount := len(nodeList.Items)
	// Check if the ResourceQuota already exists, if not create a new one
	found := &corev1.ResourceQuota{}
	err = r.Get(ctx, types.NamespacedName{Name: managedResourceQuota.Name, Namespace: managedResourceQuota.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		// Define a new ResourceQuota using nodeCount
		rq := r.resourcequotaforManagedResourceQuota(managedResourceQuota, nodeCount)
		log.Info("Creating a new ResourceQuota", "ResourceQuota.Namespace", rq.Namespace, "ResourceQuota.Name", rq.Name)
		err = r.Create(ctx, rq)
		if err != nil {
			log.Error(err, "Failed to create new ResourceQuota", "ResourceQuota.Namespace", rq.Namespace, "ResourceQuota.Name", rq.Name)
			return ctrl.Result{}, err
		}
		// ResourceQuota created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get ResourceQuota")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	hard := renderHardLimits(managedResourceQuota.Spec.Template.Hard, nodeCount)
	if !reflect.DeepEqual(found.Spec.Hard, hard) {
		log.Info("Updating ResourceQuota")
		found.Spec.Hard = hard
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update ResourceQuota", "ResourceQuota.Namespace", found.Namespace, "ResourceQuota.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, err
}

// resourcequotaforManagedResourceQuota returns a Resource Quota object
func (r *ManagedResourceQuotaReconciler) resourcequotaforManagedResourceQuota(m *k8sresourcequotaautoscalerv1beta1.ManagedResourceQuota, nodes int) *corev1.ResourceQuota {
	ls := labelsForResourceQuota(m.Name)
	hard := renderHardLimits(m.Spec.Template.Hard, nodes)

	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard:          hard,
			Scopes:        m.Spec.Template.Scopes,
			ScopeSelector: m.Spec.Template.ScopeSelector,
		},
	}
	// Set resourcequota instance as the owner and controller
	ctrl.SetControllerReference(m, rq, r.Scheme)
	return rq
}

// labelsForResourceQuota returns the labels for selecting the resources
// belonging to the given k8s-resourcequota-autoscaler CR name.
func labelsForResourceQuota(name string) map[string]string {
	return map[string]string{"app.kubernetes.io/managed-by": "k8s-resourcequota-autoscaler", "app.kubernetes.io/name": name}
}

func renderHardLimits(hard k8sresourcequotaautoscalerv1beta1.ResourceList, nodes int) map[corev1.ResourceName]resource.Quantity {
	m := map[corev1.ResourceName]resource.Quantity{}
	for key, element := range hard {
		var tpl bytes.Buffer
		t := template.Must(template.New("hard").Funcs(sprig.FuncMap()).Parse(element))
		data := struct {
			Nodes int
		}{
			Nodes: nodes,
		}
		if err := t.Execute(&tpl, data); err != nil {
			return nil
		}

		m[key], _ = resource.ParseQuantity(tpl.String())
	}
	return m
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedResourceQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sresourcequotaautoscalerv1beta1.ManagedResourceQuota{}).
		Owns(&corev1.Node{}).
		Complete(r)
}
