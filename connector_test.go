package kubernetes_test

import (
	"context"
	"io"
	v1 "k8s.io/api/core/v1"
	"os"
	"strings"
	"testing"

	"go.arcalot.io/assert"
	log "go.arcalot.io/log/v2"
	kubernetes "go.flow.arcalot.io/kubernetesdeployer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestSimpleInOut(t *testing.T) {
	configStruct := getConfigStruct(t)
	factory := kubernetes.NewFactory()
	schema := factory.ConfigurationSchema()
	serializedConfig, err := schema.SerializeType(&configStruct)
	assert.NoError(t, err)
	unserializedConfig, err := schema.UnserializeType(serializedConfig)
	assert.NoError(t, err)
	connector, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
	assert.NoError(t, err)

	container, err := connector.Deploy(context.Background(), "quay.io/joconnel/io-test-script")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, container.Close())
	})

	var containerInput = []byte("abc\n")
	assert.NoErrorR[int](t)(container.Write(containerInput))

	buf := new(strings.Builder)
	assert.NoErrorR[int64](t)(io.Copy(buf, container))
	assert.Contains(t, buf.String(), "This is what input was received: \"abc\"")

	assert.Equals(t, len(container.ID()) > 0, true)
}

func TestSecurityContextSerialization(t *testing.T) {
	boolVar := false
	var oneThousandVar int64 = 1000
	seccompProfile := v1.SeccompProfile{
		Type: v1.SeccompProfileTypeUnconfined,
	}

	containerSecurityContext := v1.SecurityContext{
		RunAsNonRoot:             &boolVar,
		RunAsUser:                &oneThousandVar,
		RunAsGroup:               &oneThousandVar,
		SeccompProfile:           &seccompProfile,
		AllowPrivilegeEscalation: &boolVar,
	}

	podSecurityContext := v1.PodSecurityContext{
		RunAsNonRoot:   &boolVar,
		RunAsUser:      &oneThousandVar,
		RunAsGroup:     &oneThousandVar,
		FSGroup:        &oneThousandVar,
		SeccompProfile: &seccompProfile,
	}

	configStruct := getConfigStruct(t)
	podSpec := kubernetes.PodSpec{
		PluginContainer: v1.Container{
			SecurityContext: &containerSecurityContext,
		},
	}
	podSpec.SecurityContext = &podSecurityContext
	configStruct.Pod = kubernetes.Pod{
		Metadata: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: podSpec,
	}

	factory := kubernetes.NewFactory()
	schema := factory.ConfigurationSchema()
	serializedConfig, err := schema.SerializeType(&configStruct)
	assert.NoError(t, err)
	_, err = schema.UnserializeType(serializedConfig)
	assert.NoError(t, err)

	// test seccomp enum
	seccompProfile.Type = "not_working"
	_, err = schema.SerializeType(&configStruct)
	assert.Error(t, err)

}

func getConfigStruct(t *testing.T) kubernetes.Config {
	dirname, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Skipping test, cannot find user home directory (%v)", err)
	}
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: dirname + "/.kube/config"},
		&clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{Server: ""}})
	kubeconfig, err := cfg.ClientConfig()
	if err != nil {
		t.Skipf("Skipping test, load kubeconfig file from user home directory (%v)", err)
	}
	namespace, _, err := cfg.Namespace()
	if err != nil {
		t.Skipf("Skipping test, load kubeconfig file from user home directory (%v)", err)
	}

	configStruct := kubernetes.Config{
		Connection: kubernetes.Connection{
			Host:        kubeconfig.Host,
			APIPath:     kubeconfig.APIPath,
			Username:    kubeconfig.Username,
			Password:    kubeconfig.Password,
			ServerName:  kubeconfig.ServerName,
			CertData:    string(kubeconfig.CertData),
			KeyData:     string(kubeconfig.KeyData),
			CAData:      string(kubeconfig.CAData),
			BearerToken: kubeconfig.BearerToken,
			QPS:         float64(kubeconfig.QPS),
			Burst:       int64(kubeconfig.Burst),
		},
		Pod: kubernetes.Pod{
			Metadata: metav1.ObjectMeta{
				Namespace: namespace,
			},
		},
	}

	return configStruct
}
