package controllers

import (
	"context"
	"errors"

	"github.com/fayvori/gitlab-operator/api/v1alpha1"
	"github.com/fayvori/gitlab-operator/internal/constants"
	gitlabapi "github.com/fayvori/gitlab-operator/internal/gitlab"
	"github.com/xanzy/go-gitlab"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	CannotGetTokenFromSecret = errors.New("Cannot get token from secret")
)

// Get gitlab client singletone, if instance not found - creates one with token from secret value.
func (r *RunnerReconciler) getGitlabClient(ctx context.Context, runnerObj *v1alpha1.Runner) (gitlabapi.GitlabClient, error) {
	if r.GitlabApi != nil {
		return *r.GitlabApi, nil
	}

	var gitlabSecret corev1.Secret
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: runnerObj.Namespace,
		Name:      constants.GitlabTokenSecretName,
	}, &gitlabSecret); err != nil {
		return nil, err
	}

	token, ok := gitlabSecret.Data["token"]

	if !ok || string(token) == "" {
		return nil, CannotGetTokenFromSecret
	}

	decryptedToken := string(token)

	var gitlabBaseUrl string = ""

	if len(runnerObj.Spec.GitlabBaseUrl) != 0 {
		gitlabBaseUrl = runnerObj.Spec.GitlabBaseUrl
	}

	client, err := gitlabapi.NewGitlabClient(decryptedToken, gitlabBaseUrl)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func (r *RunnerReconciler) RegisterNewUserRunner(ctx context.Context, runnerObj *v1alpha1.Runner) (*gitlab.UserRunner, error) {
	client, err := r.getGitlabClient(ctx, runnerObj)

	if err != nil {
		return nil, err
	}

	runnerOptions := runnerObj.Spec.RunnerOptions

	userRunnerOptions := &gitlab.CreateUserRunnerOptions{
		RunnerType:      runnerOptions.RunnerType,
		GroupID:         runnerOptions.GroupID,
		ProjectID:       runnerOptions.ProjectID,
		Description:     runnerOptions.Description,
		Paused:          runnerOptions.Paused,
		Locked:          runnerOptions.Locked,
		RunUntagged:     runnerOptions.RunUntagged,
		TagList:         runnerOptions.TagList,
		AccessLevel:     runnerOptions.AccessLevel,
		MaximumTimeout:  runnerOptions.MaximumTimeout,
		MaintenanceNote: runnerOptions.MaintenanceNote,
	}

	userRunner, _, err := client.RegisterNewUserRunner(userRunnerOptions)

	if err != nil {
		return nil, err
	}

	// TODO Enable runner for multiple projects
	for _, enableForProject := range runnerObj.Spec.EnableFor {
		projectID, err := client.GetProjectIDByPathWithNamespace(enableForProject)

		if err != nil {
			return nil, err
		}

		opt := &gitlab.EnableProjectRunnerOptions{
			RunnerID: userRunner.ID,
		}

		// Runner already enabled for this project because we are registered it as single project_type runner
		if projectID != *runnerObj.Spec.RunnerOptions.ProjectID {
			if _, err := client.EnableProjectRunner(projectID, opt); err != nil {
				return nil, err
			}
		}
	}

	return userRunner, nil
}

func (r *RunnerReconciler) UpdateRunnerDetails(ctx context.Context, rid int, runnerObj *v1alpha1.Runner) error {
	client, err := r.getGitlabClient(ctx, runnerObj)

	if err != nil {
		return err
	}

	runnerSpec := runnerObj.Spec.RunnerOptions

	opt := &gitlab.UpdateRunnerDetailsOptions{
		Description:    runnerSpec.Description,
		Paused:         runnerSpec.Paused,
		TagList:        runnerSpec.TagList,
		RunUntagged:    runnerSpec.RunUntagged,
		Locked:         runnerSpec.Locked,
		AccessLevel:    runnerSpec.AccessLevel,
		MaximumTimeout: runnerSpec.MaximumTimeout,
	}

	if err := client.UpdateRunnerDetails(rid, opt); err != nil {
		return err
	}

	return nil
}

func (r *RunnerReconciler) DeleteRunnerById(ctx context.Context, runnerObj *v1alpha1.Runner, runnerId int) error {
	client, err := r.getGitlabClient(ctx, runnerObj)

	if err != nil {
		return err
	}

	if _, err := client.DeleteRunnerById(runnerId); err != nil {
		return err
	}

	return nil
}
