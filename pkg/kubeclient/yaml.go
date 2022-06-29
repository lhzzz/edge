package kubeclient

import (
	"edge/pkg/util"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func ReadYaml(inputPath, defaults string) string {
	var yaml string = defaults
	if util.IsFileExist(inputPath) {
		yamlData, err := util.ReadFile(inputPath)
		if err != nil {
			klog.Errorf("Read yaml file: %s, error: %v", inputPath, err)
		}
		yaml = string(yamlData)
	}
	return yaml
}

func CreateByYamlFile(clientSet kubernetes.Interface, yamlFile string) error {
	err := CreateResourceWithFile(clientSet, yamlFile, nil)
	if err != nil {
		klog.Errorf("Apply yaml: %s, error: %v", yamlFile, err)
		return err
	}
	return nil
}

func DeleteByYamlFile(clientSet kubernetes.Interface, yamlFile string) error {
	err := DeleteResourceWithFile(clientSet, yamlFile, nil)
	if err != nil {
		klog.Errorf("Delete yaml: %s, error: %v", yamlFile, err)
		return err
	}
	return nil
}
