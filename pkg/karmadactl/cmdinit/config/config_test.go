/*
Copyright 2024 The Karmada Authors.

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

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testConfig = `
apiVersion: config.karmada.io/v1beta1
kind: InitConfiguration
generalConfig:
  namespace: "karmada-system"
  kubeConfigPath: "/etc/karmada/kubeconfig"
  privateImageRegistry: "local.registry.com"
  waitComponentReadyTimeout: 120
  port: 32443

certificateConfig:
  certificatesDir: "/etc/karmada/pki"
  externalDNS:
    - "www.karmada.io"
  externalIP:
    - "10.235.1.2"
  validityPeriod: "8760h"

etcdConfig:
  local:
    image: "local.registry.com/library/etcd:3.5.13-0"
    initImage: "docker.io/alpine:3.19.1"
    dataDir: "/var/lib/karmada-etcd"
    pvcSize: "5Gi"
    nodeSelectorLabels: "karmada.io/etcd=true"
    storageMode: "PVC"
    replicas: 3
  external:
    externalCAPath: "/etc/ssl/certs/ca-certificates.crt"
    externalCertPath: "/path/to/your/certificate.pem"
    externalKeyPath: "/path/to/your/privatekey.pem"
    externalServers: "https://example.com:8443"
    externalPrefix: "ext-"

controlPlaneConfig:
  apiServer:
    image: "karmada-apiserver:latest"
    advertiseAddress: "192.168.1.2"
    replicas: 3
  controllerManager:
    image: "karmada-controller-manager:latest"
    replicas: 3
  scheduler:
    image: "karmada-scheduler:latest"
    replicas: 3
  webhook:
    image: "karmada-webhook:latest"
    replicas: 3

imageConfig:
  kubeImageRegistry: "registry.cn-hangzhou.aliyuncs.com/google_containers"
  kubeImageMirrorCountry: "cn"
  imagePullPolicy: "IfNotPresent"
  imagePullSecrets:
    - "PullSecret1"
    - "PullSecret2"
`

const invalidTestConfig = `
apiVersion: config.karmada.io/v1beta1
kind: InitConfiguration
generalConfig:
  namespace: karmada-system
  kubeConfigPath: /etc/karmada/kubeconfig
  privateImageRegistry: local.registry.com
  waitComponentReadyTimeout: invalid-int
  port: 32443
`

func TestLoadInitConfiguration(t *testing.T) {
	expectedConfig := &InitConfiguration{
		GeneralConfig: GeneralConfig{
			Namespace:                 "karmada-system",
			KubeConfigPath:            "/etc/karmada/kubeconfig",
			PrivateImageRegistry:      "local.registry.com",
			WaitComponentReadyTimeout: 120,
			Port:                      32443,
		},
		CertificateConfig: CertificateConfig{
			CertificatesDir: "/etc/karmada/pki",
			ExternalDNS:     []string{"www.karmada.io"},
			ExternalIP:      []string{"10.235.1.2"},
			ValidityPeriod:  "8760h",
		},
		EtcdConfig: EtcdConfig{
			Local: &LocalEtcd{
				Image:              "local.registry.com/library/etcd:3.5.13-0",
				InitImage:          "docker.io/alpine:3.19.1",
				DataDir:            "/var/lib/karmada-etcd",
				PVCSize:            "5Gi",
				NodeSelectorLabels: "karmada.io/etcd=true",
				StorageMode:        "PVC",
				Replicas:           3,
			},
			External: &ExternalEtcd{
				ExternalCAPath:   "/etc/ssl/certs/ca-certificates.crt",
				ExternalCertPath: "/path/to/your/certificate.pem",
				ExternalKeyPath:  "/path/to/your/privatekey.pem",
				ExternalServers:  "https://example.com:8443",
				ExternalPrefix:   "ext-",
			},
		},
		ControlPlaneConfig: ControlPlaneConfig{
			APIServer: APIServerConfig{
				Image:            "karmada-apiserver:latest",
				AdvertiseAddress: "192.168.1.2",
				Replicas:         3,
			},
			ControllerManager: ControllerManagerConfig{
				Image:    "karmada-controller-manager:latest",
				Replicas: 3,
			},
			Scheduler: SchedulerConfig{
				Image:    "karmada-scheduler:latest",
				Replicas: 3,
			},
			Webhook: WebhookConfig{
				Image:    "karmada-webhook:latest",
				Replicas: 3,
			},
		},
		ImageConfig: ImageConfig{
			KubeImageRegistry:      "registry.cn-hangzhou.aliyuncs.com/google_containers",
			KubeImageMirrorCountry: "cn",
			ImagePullPolicy:        "IfNotPresent",
			ImagePullSecrets:       []string{"PullSecret1", "PullSecret2"},
		},
	}
	expectedConfig.Kind = "InitConfiguration"
	expectedConfig.APIVersion = "config.karmada.io/v1beta1"

	t.Run("Test Load Valid Configuration", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(testConfig))
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		config, err := LoadInitConfiguration(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
	})

	t.Run("Test Load Invalid Configuration", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid-config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(invalidTestConfig))
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		_, err = LoadInitConfiguration(tmpFile.Name())
		assert.Error(t, err)
	})

	t.Run("Test Load Non-Existent Configuration", func(t *testing.T) {
		_, err := LoadInitConfiguration("non-existent-file.yaml")
		assert.Error(t, err)
	})
}
