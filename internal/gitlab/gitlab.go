package gitlabapi

import (
	"errors"
	"net/http"

	"github.com/xanzy/go-gitlab"
	"k8s.io/utils/pointer"
)

var (
	RunnerAlreadyEnabled  = errors.New("Runner already enabled for this project")
	RunnerAlreadyDisabled = errors.New("Runner already disabled for this project")
	RunnerNotFound        = errors.New("Runner is not found")
)

type GitlabClient interface {
	RegisterNewUserRunner(opt *gitlab.CreateUserRunnerOptions) (*gitlab.UserRunner, *gitlab.Response, error)
	DeleteRunnerById(rid int) (*gitlab.Response, error)
	GetProjectIDByPathWithNamespace(projectPath string) (int, error)

	EnableProjectRunner(pid interface{}, opt *gitlab.EnableProjectRunnerOptions) (*gitlab.Runner, error)
	DisableProjectRunner(pid interface{}, runnerId int) (*gitlab.Response, error)
}

type gitlabApi struct {
	client *gitlab.Client
}

func (g *gitlabApi) RegisterNewUserRunner(opt *gitlab.CreateUserRunnerOptions) (*gitlab.UserRunner, *gitlab.Response, error) {
	runner, _, err := g.client.Users.CreateUserRunner(opt)

	if err != nil {
		return nil, nil, err
	}

	// Check if runner successfully registered
	opts := &gitlab.VerifyRegisteredRunnerOptions{
		Token: &runner.Token,
	}

	if response, err := g.client.Runners.VerifyRegisteredRunner(opts); err != nil || response.StatusCode != http.StatusOK {
		return nil, nil, err
	}

	return runner, nil, nil
}

func (g *gitlabApi) DeleteRunnerById(rid int) (*gitlab.Response, error) {
	// Get a list of projects which runner with id `rid` assigned to
	runnerDetails, _, err := g.client.Runners.GetRunnerDetails(rid)

	if err != nil {
		return nil, err
	}

	// Disable runner in all projects which it assigned to, due to runner delete
	for _, projects := range runnerDetails.Projects {
		g.DisableProjectRunner(projects.ID, rid)
	}

	// Delete this runner by id
	resp, err := g.client.Runners.DeleteRegisteredRunnerByID(rid)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// TODO
func (g *gitlabApi) GetProjectIDByPathWithNamespace(projectPath string) (int, error) {
	projects, _, err := g.client.Projects.ListUserProjects(11901232, &gitlab.ListProjectsOptions{
		Search:           pointer.String(projectPath),
		SearchNamespaces: pointer.Bool(true),
	})

	if err != nil {
		return 0, err
	}

	for _, project := range projects {
		return project.ID, nil
	}

	return 0, nil
}

func (g *gitlabApi) EnableProjectRunner(pid interface{}, opt *gitlab.EnableProjectRunnerOptions) (*gitlab.Runner, error) {
	runner, resp, err := g.client.Runners.EnableProjectRunner(pid, opt)

	if err != nil && resp == nil {
		return nil, err
	}

	// TODO
	switch err != nil && resp.StatusCode != http.StatusCreated {
	case resp.StatusCode == http.StatusBadRequest:
		return nil, RunnerAlreadyEnabled
	case resp.StatusCode == http.StatusNotFound:
		return nil, RunnerNotFound
	}

	return runner, nil
}

func (g *gitlabApi) DisableProjectRunner(pid interface{}, runnerId int) (*gitlab.Response, error) {
	resp, err := g.client.Runners.DisableProjectRunner(pid, runnerId)

	if err != nil && resp == nil {
		return nil, err
	}

	switch err != nil && resp.StatusCode != http.StatusNoContent {
	case resp.StatusCode == http.StatusNotFound:
		return nil, RunnerNotFound
	}

	return resp, nil
}

func NewGitlabClient(token, url string) (GitlabClient, error) {
	var err error

	if url == "" {
		url = "https://gitlab.com"
	}

	obj := &gitlabApi{}
	obj.client, err = gitlab.NewClient(token, gitlab.WithBaseURL(url))

	if err != nil {
		return nil, err
	}

	return obj, err
}
