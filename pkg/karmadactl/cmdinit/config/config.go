package config

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlserializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
	"os"
	"sort"
)

func LoadInitConfiguration(cfgPath string) (*InitConfiguration, error) {
	var config *InitConfiguration
	var err error

	config, err = loadInitConfigurationFromFile(cfgPath)

	return config, err
}

func loadInitConfigurationFromFile(cfgPath string) (*InitConfiguration, error) {
	klog.V(1).Infof("loading configuration from %q", cfgPath)

	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read config from %q: %v", cfgPath, err)
	}

	return BytesToInitConfiguration(b)
}

func BytesToInitConfiguration(b []byte) (*InitConfiguration, error) {
	gvkmap, err := SplitYAMLDocuments(b)
	if err != nil {
		return nil, err
	}

	return documentMapToInitConfiguration(gvkmap)
}

func SplitYAMLDocuments(yamlBytes []byte) (map[schema.GroupVersionKind][]byte, error) {
	gvkmap := make(map[schema.GroupVersionKind][]byte)
	knownKinds := map[string]bool{}
	errs := []error{}
	buf := bytes.NewBuffer(yamlBytes)
	reader := yaml.NewYAMLReader(bufio.NewReader(buf))
	for {
		b, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			break
		}
		gvk, err := yamlserializer.DefaultMetaFactory.Interpret(b)
		if err != nil {
			return nil, err
		}
		if len(gvk.Group) == 0 || len(gvk.Version) == 0 || len(gvk.Kind) == 0 {
			return nil, errors.Errorf("invalid configuration for GroupVersionKind %+v: kind and apiVersion is mandatory information that must be specified", gvk)
		}
		if known := knownKinds[gvk.Kind]; known {
			errs = append(errs, errors.Errorf("invalid configuration: kind %q is specified twice in YAML file", gvk.Kind))
			continue
		}
		knownKinds[gvk.Kind] = true
		gvkmap[*gvk] = b
	}
	if err := errorsutil.NewAggregate(errs); err != nil {
		return nil, err
	}
	return gvkmap, nil
}

func documentMapToInitConfiguration(gvkmap map[schema.GroupVersionKind][]byte) (*InitConfiguration, error) {
	var initcfg *InitConfiguration

	gvks := make([]schema.GroupVersionKind, 0, len(gvkmap))
	for gvk := range gvkmap {
		gvks = append(gvks, gvk)
	}
	sort.Slice(gvks, func(i, j int) bool {
		return gvks[i].String() < gvks[j].String()
	})

	for _, gvk := range gvks {
		fileContent := gvkmap[gvk]
		if gvk.Kind == "InitConfiguration" {
			initcfg = &InitConfiguration{}
			if err := yaml.Unmarshal(fileContent, initcfg); err != nil {
				return nil, err
			}
		}
	}

	if initcfg == nil {
		return nil, fmt.Errorf("no InitConfiguration kind was found in the YAML file")
	}

	return initcfg, nil
}
