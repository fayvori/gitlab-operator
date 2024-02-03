package generator

import (
	"time"

	"github.com/fayvori/gitlab-operator/api/v1alpha1"
	"github.com/pelletier/go-toml/v2"
)

// concurrent = 10
// check_interval = 30
// log_level = "info"
// shutdown_timeout = 0

// [session_server]
//   session_timeout = 1800

// [[runners]]
//   name = "gitlab-runner-fc4bdb6f9-x7wwk"
//   url = "https://gitlab.com"
//   id = 29442482
//   token = "glrt-aMyn8vsq8cyJBZoUKeZA"
//   token_obtained_at = 2023-11-11T21:38:39Z
//   token_expires_at = 0001-01-01T00:00:00Z
//   executor = "kubernetes"
//   [runners.cache]
//     MaxUploadedArchiveSize = 0
//   [runners.kubernetes]
//     host = ""
//     bearer_token_overwrite_allowed = false
//     image = "alpine"
//     namespace = "default"
//     namespace_overwrite_allowed = ""
//     node_selector_overwrite_allowed = ""
//     pod_labels_overwrite_allowed = ""
//     service_account_overwrite_allowed = ""
//     pod_annotations_overwrite_allowed = ""
//     [runners.kubernetes.pod_security_context]
//     [runners.kubernetes.init_permissions_container_security_context]
//     [runners.kubernetes.build_container_security_context]
//     [runners.kubernetes.helper_container_security_context]
//     [runners.kubernetes.service_container_security_context]
//     [runners.kubernetes.volumes]
//     [runners.kubernetes.dns_config]

// Generator is used to create Gitlab Runner config and Marshal it to toml format
type GitlabRunnerConfig struct {
	Concurrent      int           `toml:"concurrent"`
	CheckInterval   int           `toml:"check_interval"`
	LogLevel        string        `toml:"log_level"`
	ShutdownTimeout int           `toml:"shutdown_timeout"`
	SessionServer   SessionServer `toml:"session_server"`
	Runners         []Runners     `toml:"runners"`
}

type SessionServer struct {
	SessionTimeout int `toml:"session_timeout"`
}

type Cache struct {
	MaxUploadedArchiveSize int `toml:"MaxUploadedArchiveSize"`
}

// TODO
type PodSecurityContext struct{}

type InitPermissionsContainerSecurityContext struct{}

type BuildContainerSecurityContext struct{}

type HelperContainerSecurityContext struct{}

type ServiceContainerSecurityContext struct{}

type Volumes struct{}

type DNSConfig struct{}

// TODO default values
type Kubernetes struct {
	Host                                    string                                  `toml:"host"`
	BearerTokenOverwriteAllowed             bool                                    `toml:"bearer_token_overwrite_allowed"`
	Image                                   string                                  `toml:"image"`
	Namespace                               string                                  `toml:"namespace"`
	NamespaceOverwriteAllowed               string                                  `toml:"namespace_overwrite_allowed"`
	NodeSelectorOverwriteAllowed            string                                  `toml:"node_selector_overwrite_allowed"`
	PodLabelsOverwriteAllowed               string                                  `toml:"pod_labels_overwrite_allowed"`
	ServiceAccountOverwriteAllowed          string                                  `toml:"service_account_overwrite_allowed"`
	PodAnnotationsOverwriteAllowed          string                                  `toml:"pod_annotations_overwrite_allowed"`
	PodSecurityContext                      PodSecurityContext                      `toml:"pod_security_context"`
	InitPermissionsContainerSecurityContext InitPermissionsContainerSecurityContext `toml:"init_permissions_container_security_context"`
	BuildContainerSecurityContext           BuildContainerSecurityContext           `toml:"build_container_security_context"`
	HelperContainerSecurityContext          HelperContainerSecurityContext          `toml:"helper_container_security_context"`
	ServiceContainerSecurityContext         ServiceContainerSecurityContext         `toml:"service_container_security_context"`
	Volumes                                 Volumes                                 `toml:"volumes"`
	DNSConfig                               DNSConfig                               `toml:"dns_config"`
}

type Runners struct {
	Name            string     `toml:"name"`
	URL             string     `toml:"url"`
	ID              int        `toml:"id"`
	Token           string     `toml:"token"`
	TokenObtainedAt time.Time  `toml:"token_obtained_at"`
	TokenExpiresAt  *time.Time `toml:"token_expires_at"`
	Executor        string     `toml:"executor"`
	Cache           Cache      `toml:"cache"`
	Kubernetes      Kubernetes `toml:"kubernetes"`
}

type ConfigGenerator struct {
}

func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

func (gc *ConfigGenerator) Generate(runnerObj *v1alpha1.Runner, runnerID int, tokenExpiredAt *time.Time, runnerToken string) (string, error) {
	config := &GitlabRunnerConfig{
		Concurrent:      10,
		CheckInterval:   5,
		LogLevel:        "info",
		ShutdownTimeout: 0,
		SessionServer: SessionServer{
			SessionTimeout: 1800,
		},
		Runners: []Runners{
			Runners{
				Name:            runnerObj.Name,
				URL:             "https://gitlab.com",
				ID:              runnerID,
				Token:           runnerToken,
				TokenObtainedAt: time.Now(),
				TokenExpiresAt:  tokenExpiredAt,
				Executor:        "kubernetes",
				Cache: Cache{
					MaxUploadedArchiveSize: 0,
				},
				// TODO
				Kubernetes: Kubernetes{
					// TODO
					Host:                                    "replace",
					BearerTokenOverwriteAllowed:             false,
					Image:                                   "alpine",
					Namespace:                               runnerObj.Namespace,
					NamespaceOverwriteAllowed:               "",
					NodeSelectorOverwriteAllowed:            "",
					PodLabelsOverwriteAllowed:               "",
					ServiceAccountOverwriteAllowed:          "",
					PodAnnotationsOverwriteAllowed:          "",
					PodSecurityContext:                      PodSecurityContext{},
					InitPermissionsContainerSecurityContext: InitPermissionsContainerSecurityContext{},
					BuildContainerSecurityContext:           BuildContainerSecurityContext{},
					HelperContainerSecurityContext:          HelperContainerSecurityContext{},
					ServiceContainerSecurityContext:         ServiceContainerSecurityContext{},
					Volumes:                                 Volumes{},
					DNSConfig:                               DNSConfig{},
				},
			},
		},
	}

	toml, err := toml.Marshal(config)

	if err != nil {
		return "", err
	}

	return string(toml), nil
}
