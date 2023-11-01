package kubernetes

import (
	"fmt"

	log "go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/schema"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
)

// NewFactory creates a new factory for the Docker deployer.
func NewFactory() deployer.ConnectorFactory[*Config] {
	return &factory{}
}

type factory struct {
}

func (f factory) Name() string {
	return "kubernetes"
}

func (f factory) DeploymentType() deployer.DeploymentType {
	return "image"
}

func (f factory) ConfigurationSchema() *schema.TypedScopeSchema[*Config] {
	return Schema
}

func (f factory) Create(config *Config, logger log.Logger) (deployer.Connector, error) {
	connectionConfig := f.createConnectionConfig(config)

	cli, err := kubernetes.NewForConfig(&connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes config (%w)", err)
	}

	restClient, err := restclient.RESTClientFor(&connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes REST client (%w)", err)
	}

	return &connector{
		cli:              cli,
		restClient:       restClient,
		config:           config,
		connectionConfig: connectionConfig,
		logger:           logger,
	}, nil
}

func (f factory) createConnectionConfig(config *Config) restclient.Config {
	return restclient.Config{
		Host:    config.Connection.Host,
		APIPath: config.Connection.APIPath,
		ContentConfig: restclient.ContentConfig{
			GroupVersion:         &core.SchemeGroupVersion,
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		},
		Username:    config.Connection.Username,
		Password:    config.Connection.Password,
		BearerToken: config.Connection.BearerToken,
		Impersonate: restclient.ImpersonationConfig{},
		TLSClientConfig: restclient.TLSClientConfig{
			ServerName: config.Connection.ServerName,
			CertData:   []byte(config.Connection.CertData),
			KeyData:    []byte(config.Connection.KeyData),
			CAData:     []byte(config.Connection.CAData),
		},
		UserAgent: "Arcaflow",
		QPS:       float32(config.Connection.QPS),
		Burst:     int(config.Connection.Burst),
		Timeout:   config.Timeouts.HTTP,
	}
}
