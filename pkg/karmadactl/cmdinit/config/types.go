package config

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type InitConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// GeneralConfig Contains the common configuration at initialization.
	GeneralConfig GeneralConfig `yaml:"generalConfig"`

	// CertificateConfig Contains configurations related to certificates.
	CertificateConfig CertificateConfig `yaml:"certificateConfig"`

	// EtcdConfig Contains configuration for etcd.
	EtcdConfig EtcdConfig `yaml:"etcdConfig"`

	// ControlPlaneConfig Contains control plane configurations.
	ControlPlaneConfig ControlPlaneConfig `yaml:"controlPlaneConfig"`

	// ImageConfig Contains image-related configurations.
	ImageConfig ImageConfig `yaml:"imageConfig"`
}

type GeneralConfig struct {
	Namespace                 string `yaml:"namespace"`
	KubeConfigPath            string `yaml:"kubeConfigPath"`
	PrivateImageRegistry      string `yaml:"privateImageRegistry"`
	WaitComponentReadyTimeout int    `yaml:"waitComponentReadyTimeout"`
	Port                      int    `yaml:"port"`
}

type CertificateConfig struct {
	CertificatesDir string   `yaml:"certificatesDir"`
	ExternalDNS     []string `yaml:"externalDNS"`
	ExternalIP      []string `yaml:"externalIP"`
	ValidityPeriod  string   `yaml:"validityPeriod"`
	ExtraArgs       []Arg    `yaml:"extraArgs"`
}

type EtcdConfig struct {
	Local    *LocalEtcd    `json:"local,omitempty"`
	External *ExternalEtcd `json:"external,omitempty"`
}

type LocalEtcd struct {
	Image              string `yaml:"image"`
	InitImage          string `yaml:"initImage"`
	DataDir            string `yaml:"dataDir"`
	PVCSize            string `yaml:"pvcSize"`
	NodeSelectorLabels string `yaml:"nodeSelectorLabels"`
	StorageMode        string `yaml:"storageMode"`
	Replicas           int32  `yaml:"replicas"`
	ExtraArgs          []Arg  `yaml:"extraArgs"`
}

type ExternalEtcd struct {
	ExternalCAPath   string `yaml:"externalCAPath"`
	ExternalCertPath string `yaml:"externalCertPath"`
	ExternalKeyPath  string `yaml:"externalKeyPath"`
	ExternalServers  string `yaml:"externalServers"`
	ExternalPrefix   string `yaml:"externalPrefix"`
	ExtraArgs        []Arg  `yaml:"extraArgs"`
}

type ControlPlaneConfig struct {
	APIServer         APIServerConfig         `yaml:"apiServer"`
	ControllerManager ControllerManagerConfig `yaml:"controllerManager"`
	Scheduler         SchedulerConfig         `yaml:"scheduler"`
	Webhook           WebhookConfig           `yaml:"webhook"`
}

type APIServerConfig struct {
	Image            string `yaml:"image"`
	AdvertiseAddress string `yaml:"advertiseAddress"`
	Replicas         int32  `yaml:"replicas"`
	ExtraArgs        []Arg  `yaml:"extraArgs"`
}

type ControllerManagerConfig struct {
	Image     string `yaml:"image"`
	Replicas  int32  `yaml:"replicas"`
	ExtraArgs []Arg  `yaml:"extraArgs"`
}

type SchedulerConfig struct {
	Image     string `yaml:"image"`
	Replicas  int32  `yaml:"replicas"`
	ExtraArgs []Arg  `yaml:"extraArgs"`
}

type WebhookConfig struct {
	Image     string `yaml:"image"`
	Replicas  int32  `yaml:"replicas"`
	ExtraArgs []Arg  `yaml:"extraArgs"`
}

type ImageConfig struct {
	KubeImageRegistry      string   `yaml:"kubeImageRegistry"`
	KubeImageMirrorCountry string   `yaml:"kubeImageMirrorCountry"`
	ImagePullPolicy        string   `yaml:"imagePullPolicy"`
	ImagePullSecrets       []string `yaml:"imagePullSecrets"`
}

type Arg struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
