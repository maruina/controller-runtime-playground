/*
Copyright 2024.

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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ExampleReconciler reconciles a Example object
type ExampleReconciler struct {
	client.Client
	Scheme                     *runtime.Scheme
	immediateReconcileRequests chan event.GenericEvent
}

//+kubebuilder:rbac:groups=maruina.xyz.maruina.xyz,resources=examples,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=maruina.xyz.maruina.xyz,resources=examples/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=maruina.xyz.maruina.xyz,resources=examples/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Example object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *ExampleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("reconciliation loop started", "request", req.NamespacedName)

	ns := corev1.Namespace{}
	if err := r.Get(ctx, req.NamespacedName, &ns); err != nil {
		return ctrl.Result{}, err
	}

	if ns.Name == metav1.NamespaceSystem {
		if _, ok := ns.Annotations["opt-out"]; ok {
			l.Info("system namespace found with opt-out annotation", "name", ns.Name, "namespace", ns.Namespace)
			namespaces := corev1.NamespaceList{}
			if err := r.List(ctx, &namespaces); err != nil {
				l.Error(err, "error listing namespaces")
				return ctrl.Result{}, err
			}
			for _, n := range namespaces.Items {
				if n.Name != metav1.NamespaceSystem {
					l.Info("injecting reconciliation loop for namespace", "name", n.Name, "namespace", n.Namespace)
					r.immediateReconcileRequests <- event.GenericEvent{Object: &n}
				}
			}
			return ctrl.Result{}, nil
		}
		l.Info("system namespace found", "name", ns.Name, "namespace", ns.Namespace)
	}

	l.Info("namespace found", "name", ns.Name, "namespace", ns.Namespace)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExampleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.immediateReconcileRequests = make(chan event.GenericEvent)
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Watches(
			&corev1.ResourceQuota{},
			handler.EnqueueRequestsFromMapFunc(enqueueNamespaceFromObject),
			builder.WithPredicates(ignoreUpdatePredicate()),
		).
		WatchesRawSource(source.Channel(r.immediateReconcileRequests, &handler.EnqueueRequestForObject{})).
		Complete(r)
}

// enqueueNamespaceFromObject will return a reconcile request for the namespace of the given object
func enqueueNamespaceFromObject(_ context.Context, o client.Object) []reconcile.Request {
	return []reconcile.Request{{NamespacedName: client.ObjectKey{Namespace: "", Name: o.GetNamespace()}}}
}

// ignoreUpdatePredicate will ignore update events
func ignoreUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
	}
}
