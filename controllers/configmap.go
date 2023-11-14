package controllers

import (
	"context"
	"strconv"

	"github.com/fayvori/gitlab-operator/api/v1alpha1"
	"github.com/fayvori/gitlab-operator/internal/constants"
	"github.com/fayvori/gitlab-operator/internal/generator"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *RunnerReconciler) defineConfigMap(runnerObj *v1alpha1.Runner, runnerID string, gitlabRunnerTomlConfig string) *apiv1.ConfigMap {
	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.GetConfigMapName(runnerObj.Name),
			Namespace: runnerObj.Namespace,
		},
		Immutable: pointer.Bool(false),
		Data: map[string]string{
			constants.GitlabConfigKey:   gitlabRunnerTomlConfig,
			constants.GitlabRunnerIDKey: runnerID,
		},
	}

	ctrl.SetControllerReference(runnerObj, configMap, r.Scheme)

	return configMap
}

func (r *RunnerReconciler) reconcileConfigMap(ctx context.Context, runnerObj *v1alpha1.Runner) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)

	configMapName := constants.GetConfigMapName(runnerObj.Name)

	configMap := &apiv1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: runnerObj.Namespace, Name: configMapName}, configMap)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Register new runner with GitlabApi
			userRunner, err := r.RegisterNewUserRunner(ctx, runnerObj)

			if err != nil {
				logger.Error(err, "Unable to create new runner GitlabApi")
			}

			generator := generator.NewConfigGenerator()

			generatedConfig, err := generator.Generate(runnerObj, userRunner.ID, userRunner.TokenExpiresAt, userRunner.Token)

			if err != nil {
				logger.Error(err, "Unable to marshal toml config", "invalid_config", generatedConfig)
				return &ctrl.Result{}, err
			}

			configMapDefenition := r.defineConfigMap(runnerObj, strconv.Itoa(userRunner.ID), generatedConfig)

			logger.Info("ConfigMap resource not found. Creating or re-creating it")
			err = r.Create(ctx, configMapDefenition)

			if err != nil {
				logger.Info("Failed to create ConfigMap resource. Re-running reconcile.")
				return &ctrl.Result{}, err
			}
		} else {
			logger.Info("Failed to get ConfigMap resource. Re-running reconcile.")
			return &ctrl.Result{}, err
		}
	} else {
		// TODO config update ConfigMap checksum
	}

	return &ctrl.Result{}, nil
}
