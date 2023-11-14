package constants

import "fmt"

const (
	ConfigMapNamePattern  = "%s-cm"
	GitlabTokenSecretName = "gitlab-secret"

	GitlabConfigKey      = "config.toml"
	GitlabRunnerIDKey    = "runnerID"
	GitlabRunnerTokenKey = "runnerToken"

	GitlabOperatorFinalizer = "gitlab.fayvori.gitlab-operator/finalizer"
)

// Custom statuses for operator
const (
	TypeAliveRunner        = "Alive"
	TypeProvisioningRunner = "Provisioning"
	TypeDestroingRunner    = "Destroing"
)

func GetConfigMapName(name string) string {
	return fmt.Sprintf(ConfigMapNamePattern, name)
}
