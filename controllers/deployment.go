package controllers

import (
	"context"
	"strconv"

	"reflect"

	"github.com/fayvori/gitlab-operator/api/v1alpha1"
	"github.com/fayvori/gitlab-operator/internal/constants"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *RunnerReconciler) defineDeployment(runnerObj *v1alpha1.Runner) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runnerObj.Name,
			Namespace: runnerObj.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: runnerObj.Labels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: runnerObj.Labels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						apiv1.Container{
							Args: []string{
								"run",
							},
							Name:            runnerObj.Name,
							Image:           "gitlab/gitlab-runner:alpine3.19-bleeding",
							ImagePullPolicy: apiv1.PullAlways,
							VolumeMounts: []apiv1.VolumeMount{
								apiv1.VolumeMount{
									Name:      "config",
									MountPath: "/etc/gitlab-runner/config.toml",
									SubPath:   "config.toml",
								},
							},
						},
					},
					ServiceAccountName: "gitlab-operator-controller-manager",
					Volumes: []apiv1.Volume{
						apiv1.Volume{
							Name: "config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: constants.GetConfigMapName(runnerObj.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(runnerObj, deployment, r.Scheme)

	return deployment
}

func (r *RunnerReconciler) reconcileDeployment(ctx context.Context, runnerObj *v1alpha1.Runner) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	deploymentDefenition := r.defineDeployment(runnerObj)

	deployment := &appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: runnerObj.Namespace, Name: runnerObj.Name}, deployment)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create Deployment
			logger.Info("Deployment resource not found. Creating or re-creating it")
			err = r.Create(ctx, deploymentDefenition)

			if err != nil {
				logger.Info("Failed to create Deployment resource. Re-running reconcile.")
				return &ctrl.Result{}, err
			}

			// Set condition to alive if there's no errors
			meta.SetStatusCondition(&runnerObj.Status.Conditions,
				metav1.Condition{
					Type:    constants.TypeAliveRunner,
					Status:  metav1.ConditionTrue,
					Reason:  "RunnerAlive",
					Message: "Runner is alive",
				})

			if err := r.Status().Update(ctx, runnerObj); err != nil {
				logger.Error(err, "Failed to update Runner status")
				return &ctrl.Result{}, err
			}
		} else {
			logger.Info("Failed to get Deployment resource. Re-running reconcile.")
			return &ctrl.Result{}, err
		}
	} else {
		desiredDeployment := r.defineDeployment(runnerObj)

		if !reflect.DeepEqual(deployment.Spec.DeepCopy(), desiredDeployment.Spec) {
			logger.Info("Updating Deployment to desired state")
			desiredDeployment.Spec.DeepCopyInto(&deployment.Spec)

			if r.Update(ctx, deployment); err != nil {
				logger.Error(err, "Failed to update Deployment")
				return &ctrl.Result{}, nil
			}

			foundConfigMap := &apiv1.ConfigMap{}
			err := r.Get(ctx, types.NamespacedName{Namespace: runnerObj.Namespace, Name: constants.GetConfigMapName(runnerObj.Name)}, foundConfigMap)

			if err != nil {
				logger.Error(err, "Unable to get runner configmap")
				return &ctrl.Result{}, err
			}

			runnerID, _ := strconv.Atoi(foundConfigMap.Data["runnerID"])
			err = r.UpdateRunnerDetails(ctx, runnerID, runnerObj)

			if err != nil {
				return &ctrl.Result{}, err
			}
		}
	}

	return &ctrl.Result{}, nil
}
