/*
Copyright 2021 The Karmada Authors.

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

package cert

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	"github.com/karmada-io/karmada/pkg/karmadactl/cmdinit/options"
	"github.com/karmada-io/karmada/pkg/karmadactl/cmdinit/utils"
	"github.com/karmada-io/karmada/pkg/util/names"
)

const (
	TestCertsTmp        = "./test-certs-tmp-without-ca-certificate"        //nolint
	TestCertsTmpWithArg = "./test-certs-tmp-with-ca-certificate"           //nolint
	TestCaCertPath      = "./test-certs-tmp-without-ca-certificate/ca.crt" //nolint
	TestCaKeyPath       = "./test-certs-tmp-without-ca-certificate/ca.key" //nolint
)

var certFiles = []string{
	"apiserver.crt", "apiserver.key",
	"ca.crt", "ca.key",
	"etcd-ca.crt", "etcd-ca.key",
	"etcd-client.crt", "etcd-client.key",
	"etcd-server.crt", "etcd-server.key",
	"front-proxy-ca.crt", "front-proxy-ca.key",
	"front-proxy-client.crt", "front-proxy-client.key",
	"karmada.crt", "karmada.key",
}

func TestGenCerts(t *testing.T) {
	defer os.RemoveAll(TestCertsTmp)
	defer os.RemoveAll(TestCertsTmpWithArg)

	notAfter := time.Now().Add(Duration365d * 10).UTC()
	namespace := "kube-karmada"
	flagsExternalIP := ""
	masterIP := "192.168.1.1,192.168.1.2"

	var etcdServerCertDNS = []string{
		"localhost",
	}

	for i := 0; i < 3; i++ {
		etcdServerCertDNS = append(etcdServerCertDNS, fmt.Sprintf("%s-%v.%s.%s.svc.cluster.local", "etcd", i, "etcd", namespace))
	}

	etcdServerAltNames := certutil.AltNames{
		DNSNames: etcdServerCertDNS,
		IPs:      []net.IP{utils.StringToNetIP("127.0.0.1")},
	}
	etcdServerCertConfig := NewCertConfig("karmada-etcd-server", []string{}, etcdServerAltNames, &notAfter)

	etcdClientCertCfg := NewCertConfig("karmada-etcd-client", []string{}, certutil.AltNames{}, &notAfter)

	var karmadaDNS = []string{
		"localhost",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"karmada-apiserver",
		names.KarmadaWebhookComponentName,
		fmt.Sprintf("%s.%s.svc.cluster.local", "karmada-apiserver", namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", names.KarmadaWebhookComponentName, namespace),
		fmt.Sprintf("*.%s.svc.cluster.local", namespace),
		fmt.Sprintf("*.%s.svc", namespace),
	}
	if hostName, err := os.Hostname(); err != nil {
		klog.Warningf("Failed to get the current hostname, error message: %s.", err)
	} else {
		karmadaDNS = append(karmadaDNS, hostName)
	}

	ips := utils.FlagsIP(flagsExternalIP)
	ips = append(ips, utils.FlagsIP(masterIP)...)

	internetIP, err := utils.InternetIP()
	if err != nil {
		klog.Warningf("Failed to obtain internet IP, error message: %s.", err)
	} else {
		ips = append(ips, internetIP)
	}

	ips = append(
		ips,
		utils.StringToNetIP("127.0.0.1"),
		utils.StringToNetIP("10.254.0.1"),
	)

	fmt.Println("karmada certificate ip ", ips)

	karmadaAltNames := certutil.AltNames{
		DNSNames: karmadaDNS,
		IPs:      ips,
	}

	karmadaCertCfg := NewCertConfig("system:admin", []string{"system:masters"}, karmadaAltNames, &notAfter)
	apiserverCertCfg := NewCertConfig("karmada-apiserver", []string{""}, karmadaAltNames, &notAfter)
	frontProxyClientCertCfg := NewCertConfig("front-proxy-client", []string{}, certutil.AltNames{}, &notAfter)

	if err := GenCerts(TestCertsTmp, "", "", etcdServerCertConfig, etcdClientCertCfg, karmadaCertCfg, apiserverCertCfg, frontProxyClientCertCfg); err != nil {
		t.Fatal(err)
	}
	if err := checkCertFiles(TestCertsTmp, certFiles); err != nil {
		t.Fatal(err)
	} else {
		klog.Infof("All certificate files are present without CA certificates address parameter exists")
	}

	if err := GenCerts(TestCertsTmpWithArg, TestCaCertPath, TestCaKeyPath, etcdServerCertConfig, etcdClientCertCfg, karmadaCertCfg, apiserverCertCfg, frontProxyClientCertCfg); err != nil {
		t.Fatal(err)
	}
	if err := checkCertFiles(TestCertsTmpWithArg, certFiles); err != nil {
		t.Fatal(err)
	} else {
		klog.Infof("All certificate files are present with CA certificates address parameter exists")
	}

	if ok, err := compareCertFilesInDirs(TestCertsTmp, TestCertsTmpWithArg, "ca.crt"); !ok || err != nil {
		t.Fatal(err)
	} else {
		klog.Infof("The certificate files in the two directories are the same")
	}
}

func checkCertFiles(path string, files []string) error {
	for _, file := range files {
		filePath := filepath.Join(path, file)
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("cert file not found: %s,error message: %s", filePath, err.Error())
		}
	}
	return nil
}

// compareFiles compares two files to see if they are the same
func compareFiles(file1, file2 string) (bool, error) {
	f1, err := os.Open(file1)
	if err != nil {
		return false, fmt.Errorf("failed to open file %s: %v", file1, err)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		return false, fmt.Errorf("failed to open file %s: %v", file2, err)
	}
	defer f2.Close()

	hash1 := sha256.New()
	hash2 := sha256.New()

	if _, err := io.Copy(hash1, f1); err != nil {
		return false, fmt.Errorf("failed to read file %s: %v", file1, err)
	}
	if _, err := io.Copy(hash2, f2); err != nil {
		return false, fmt.Errorf("failed to read file %s: %v", file2, err)
	}

	return string(hash1.Sum(nil)) == string(hash2.Sum(nil)), nil
}

// compareCertFilesInDirs compares specific files in two directories to check if they are the same
func compareCertFilesInDirs(dir1, dir2, filename string) (bool, error) {
	file1 := filepath.Join(dir1, filename)
	file2 := filepath.Join(dir2, filename)
	return compareFiles(file1, file2)
}

// TestNewGenCerts_FullSet_Issuer verifies that NewGenCerts generates the full
// set of certificates akin to kubernetes.Init(), and more importantly, that
// etcd and front-proxy client certificates are signed by their specific CAs
// (etcd-ca and front-proxy-ca), while the rest are signed by the main karmada CA.
func TestNewGenCerts_FullSet_Issuer(t *testing.T) { //nolint:funlen
	tmpDir := "./test-new-gen-certs-full"
	defer os.RemoveAll(tmpDir)

	notAfter := time.Now().Add(Duration365d * 2).UTC()

	// Build AltNames helpers similar to kubernetes.Init()
	namespace := names.NamespaceKarmadaSystem
	hostClusterDomain := "cluster.local"
	externalDNS := "example.com"
	externalIP := "1.2.3.4"
	karmadaAPIServerIPs := []net.IP{utils.StringToNetIP("1.2.3.5")}

	buildDefaultAltNames := func(componentName string) certutil.AltNames {
		dns := []string{
			fmt.Sprintf("%s.karmada-system.svc", componentName),
			fmt.Sprintf("%s.karmada-system.svc.cluster.local", componentName),
			fmt.Sprintf("%s.%s.svc.%s", componentName, namespace, hostClusterDomain),
			"localhost",
		}
		dns = append(dns, utils.FlagsDNS(externalDNS)...)

		ips := []net.IP{utils.StringToNetIP("127.0.0.1")}
		ips = append(ips, utils.FlagsIP(externalIP)...)
		ips = append(ips, karmadaAPIServerIPs...)
		if ip, err := utils.InternetIP(); err == nil {
			ips = append(ips, ip)
		}
		return certutil.AltNames{DNSNames: dns, IPs: ips}
	}

	// etcd server SANs (for 3 replicas) similar to kubernetes buildEtcdCertConfig
	buildEtcdServerAltNames := func(replicas int) certutil.AltNames {
		dns := []string{"localhost"}
		for i := 0; i < replicas; i++ {
			dns = append(dns, fmt.Sprintf("%s-%d.%s.%s.svc.%s", "etcd", i, "etcd", namespace, hostClusterDomain))
		}
		return certutil.AltNames{DNSNames: dns, IPs: []net.IP{utils.StringToNetIP("127.0.0.1")}}
	}

	// Compose the full cert config map akin to kubernetes.Init()
	certConfigMap := map[string]*CertsConfig{}

	// etcd
	certConfigMap[options.EtcdServerCertAndKeyName] = NewCertConfig(options.KarmadaEtcdServerCN, []string{}, buildEtcdServerAltNames(3), &notAfter)
	certConfigMap[options.EtcdClientCertAndKeyName] = NewCertConfig(options.KarmadaEtcdClientCN, []string{""}, certutil.AltNames{}, &notAfter)

	// karmada-apiserver
	certConfigMap[options.KarmadaApiServerCertAndKeyName] = NewCertConfig(options.KarmadaApiServerCN, []string{}, buildDefaultAltNames("karmada-apiserver"), &notAfter)
	certConfigMap[options.KarmadaApiServerEtcdClientCertAndKeyName] = NewCertConfig(options.KarmadaApiServerEtcdClientCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)
	certConfigMap[options.FrontProxyClientCertAndKeyName] = NewCertConfig(options.KarmadaFrontProxyClientCN, []string{}, certutil.AltNames{}, &notAfter)

	// aggregated-apiserver
	certConfigMap[options.KarmadaAggregatedApiServerCertAndKeyName] = NewCertConfig(options.KarmadaAggregatedApiServerCN, []string{}, buildDefaultAltNames(names.KarmadaAggregatedAPIServerComponentName), &notAfter)
	certConfigMap[options.KarmadaAggregatedApiServerClientCertAndKeyName] = NewCertConfig(options.KarmadaAggregatedApiServerCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)
	certConfigMap[options.KarmadaAggregatedApiServerEtcdClientCertAndKeyName] = NewCertConfig(options.KarmadaAggregatedApiServerEtcdClientCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)

	// webhook
	certConfigMap[options.KarmadaWebhookCertAndKeyName] = NewCertConfig(options.KarmadaWebhookCN, []string{}, buildDefaultAltNames(names.KarmadaWebhookComponentName), &notAfter)
	certConfigMap[options.KarmadaWebhookClientCertAndKeyName] = NewCertConfig(options.KarmadaWebhookCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)

	// search
	certConfigMap[options.KarmadaSearchCertAndKeyName] = NewCertConfig(options.KarmadaSearchCN, []string{}, buildDefaultAltNames(names.KarmadaSearchComponentName), &notAfter)
	certConfigMap[options.KarmadaSearchClientCertAndKeyName] = NewCertConfig(options.KarmadaSearchCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)
	certConfigMap[options.KarmadaSearchEtcdClientCertAndKeyName] = NewCertConfig(options.KarmadaSearchEtcdClientCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)

	// controller-manager client
	certConfigMap[options.KarmadaControllerManagerClientCertAndKeyName] = NewCertConfig(options.KarmadaControllerManagerCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)

	// scheduler (client + grpc)
	certConfigMap[options.KarmadaSchedulerGrpcCertAndKeyName] = NewCertConfig(options.KarmadaSchedulerGrpcCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)
	certConfigMap[options.KarmadaSchedulerClientCertAndKeyName] = NewCertConfig(options.KarmadaSchedulerCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)

	// descheduler (client + grpc)
	certConfigMap[options.KarmadaDeschedulerClientCertAndKeyName] = NewCertConfig(options.KarmadaDeschedulerCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)
	certConfigMap[options.KarmadaDeschedulerGrpcCertAndKeyName] = NewCertConfig(options.KarmadaDeschedulerGrpcCN, []string{"system:masters"}, certutil.AltNames{}, &notAfter)

	if err := NewGenCerts(tmpDir, "", "", certConfigMap); err != nil {
		t.Fatalf("NewGenCerts failed: %v", err)
	}

	// Expected basic CA files
	caFiles := []string{
		"ca.crt", "ca.key",
		"front-proxy-ca.crt", "front-proxy-ca.key",
		"etcd-ca.crt", "etcd-ca.key",
	}
	if err := checkCertFiles(tmpDir, caFiles); err != nil {
		t.Fatal(err)
	}

	// Validate each generated cert exists and is issued by the correct CA
	expectedIssuer := func(name string) string {
		switch name {
		case options.EtcdServerCertAndKeyName,
			options.EtcdClientCertAndKeyName,
			options.KarmadaApiServerEtcdClientCertAndKeyName,
			options.KarmadaAggregatedApiServerEtcdClientCertAndKeyName,
			options.KarmadaSearchEtcdClientCertAndKeyName:
			return options.EtcdCaCertAndKeyName
		case options.FrontProxyClientCertAndKeyName:
			return options.FrontProxyCaCertAndKeyName
		default:
			// fallback main CA created by getCACertAndKey() uses CN "karmada"
			return "karmada"
		}
	}

	for name := range certConfigMap {
		// files exist
		if err := checkCertFiles(tmpDir, []string{fmt.Sprintf("%s.crt", name), fmt.Sprintf("%s.key", name)}); err != nil {
			t.Fatalf("expected cert/key for %s missing: %v", name, err)
		}
		// issuer correctness
		issuerCN, err := readCertIssuerCN(filepath.Join(tmpDir, fmt.Sprintf("%s.crt", name)))
		if err != nil {
			t.Fatalf("parse cert %s failed: %v", name, err)
		}
		want := expectedIssuer(name)
		if issuerCN != want {
			t.Fatalf("issuer mismatch for %s: got %s, want %s", name, issuerCN, want)
		}
	}
}

// readCertIssuerCN reads a PEM certificate and returns Issuer.CommonName
func readCertIssuerCN(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM for %s", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	return cert.Issuer.CommonName, nil
}
