/*
Copyright 2023.

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
	"context"
	"fmt"
	"github.com/fayvori/gitlab-operator/api/v1alpha1"
	mydomainv1alpha1 "github.com/fayvori/gitlab-operator/api/v1alpha1"
	"github.com/fayvori/gitlab-operator/internal/constants"
	gitlabapi "github.com/fayvori/gitlab-operator/internal/gitlab"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
)

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	GitlabApi *gitlabapi.GitlabClient
	Scheme    *runtime.Scheme
}

// Validation

// +kubebuilder:rbac:groups=my.domain,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=my.domain,resources=runners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=my.domain,resources=runners/finalizers,verbs=update
func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	runnerObj := &v1alpha1.Runner{}
	err := r.Client.Get(ctx, req.NamespacedName, runnerObj)

	logger := log.FromContext(ctx).WithValues("name", runnerObj.Name, "namespace", runnerObj.Namespace)

	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Runner resource not found. Ignoring since object must be deleted.")

			return ctrl.Result{}, nil
		}

		logger.Info("Failed to get Runner resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	// TODO update runner
	// defer func() {
	// 	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
	// 		var newRunner v1alpha1.Runner

	// 		err = r.Client.Get(
	// 			ctx,
	// 			client.ObjectKey{Namespace: runnerObj.Namespace, Name: runnerObj.GetName()},
	// 			&newRunner)

	// 		switch {
	// 		case err != nil:
	// 			logger.Error(err, "cannot get runner")
	// 			return err

	// 		// no changes in status detected
	// 		case reflect.DeepEqual(runnerObj.Spec, newRunner.Spec):
	// 			return nil
	// 		}

	// 		newRunner.Spec = runnerObj.Spec

	// 		return r.Update(ctx, &newRunner)
	// 	})

	// 	if err != nil {
	// 		logger.Error(err, "cannot update runner's spec")
	// 	}
	// }()

	logger.Info("Reconciling started for Runner CRD")

	if runnerObj.Status.Conditions == nil || len(runnerObj.Status.Conditions) == 0 {
		meta.SetStatusCondition(&runnerObj.Status.Conditions,
			metav1.Condition{
				Type:    constants.TypeProvisioningRunner,
				Status:  metav1.ConditionUnknown,
				Reason:  "Reconciling",
				Message: "Provisioning runner",
			})

		if err := r.Status().Update(ctx, runnerObj); err != nil {
			logger.Error(err, "Failed to update Runner status")
			return ctrl.Result{}, err
		}
	}

	if reconcileResult, err := r.reconcileConfigMap(ctx, runnerObj); err != nil {
		return *reconcileResult, err
	}

	if reconcileResult, err := r.reconcileDeployment(ctx, runnerObj); err != nil {
		return *reconcileResult, err
	}

	// Add finalizer if it doesn't exists
	if !controllerutil.ContainsFinalizer(runnerObj, constants.GitlabOperatorFinalizer) {
		logger.Info("Adding finalizer to Runner object")
		if ok := controllerutil.AddFinalizer(runnerObj, constants.GitlabOperatorFinalizer); !ok {
			logger.Error(err, "Unable to add finalizer to the Runner object")
			return ctrl.Result{Requeue: true}, err
		}

		if err = r.Update(ctx, runnerObj); err != nil {
			logger.Error(err, "Cannot add finalizer to the resource")
			return ctrl.Result{}, err
		}
	}

	isRunnerMarkedToBeDeleted := runnerObj.GetDeletionTimestamp() != nil
	if isRunnerMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(runnerObj, constants.GitlabOperatorFinalizer) {
			logger.Info("Performing Finalizer Operations for Runner before delete CR")

			meta.SetStatusCondition(&runnerObj.Status.Conditions, metav1.Condition{Type: constants.TypeDestroingRunner,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", runnerObj.Name)})

			if err := r.Status().Update(ctx, runnerObj); err != nil {
				logger.Error(err, "Failed to update Runner status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before remove the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			r.finalizeRunner(ctx, runnerObj)

			if err := r.Get(ctx, req.NamespacedName, runnerObj); err != nil {
				logger.Error(err, "Failed to re-fetch runner")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&runnerObj.Status.Conditions, metav1.Condition{Type: constants.TypeDestroingRunner,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", runnerObj.Name)})

			if err := r.Status().Update(ctx, runnerObj); err != nil {
				logger.Error(err, "Failed to update Runner status")
				return ctrl.Result{}, err
			}

			logger.Info("Removing Finalizer for Runner after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(runnerObj, constants.GitlabOperatorFinalizer); !ok {
				logger.Error(err, "Failed to remove finalizer for Runner")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, runnerObj); err != nil {
				logger.Error(err, "Failed to remove finalizer for Runner")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) finalizeRunner(ctx context.Context, runnerObj *v1alpha1.Runner) error {
	logger := log.FromContext(ctx)

	logger.Info("Finalizing runnerObj")

	meta.SetStatusCondition(&runnerObj.Status.Conditions,
		metav1.Condition{
			Type:    constants.TypeDestroingRunner,
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting runner",
		})

	if err := r.Status().Update(ctx, runnerObj); err != nil {
		logger.Error(err, "Failed to update Runner status")
		return err
	}

	foundConfigMap := &apiv1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Namespace: runnerObj.Namespace, Name: constants.GetConfigMapName(runnerObj.Name)}, foundConfigMap)

	if err != nil {
		logger.Error(err, "Unable to get runner configmap")
		return err
	}

	runnerID, _ := strconv.Atoi(foundConfigMap.Data["runnerID"])
	err = r.DeleteRunnerById(ctx, runnerObj, runnerID)

	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1alpha1.Runner{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
