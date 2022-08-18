package server

import (
	"context"
	"edge/internal/constant"
	"edge/internal/constant/manifests"
	"edge/pkg/kubeclient"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createEdgeNode(ctx context.Context, nodeName string) (bool, error) {
	option := map[string]string{
		"NodeName": nodeName,
	}
	isNeedCreate := false
	_, err := k8sClient().CoreV1().Nodes().Get(context.TODO(), nodeName, v1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		isNeedCreate = true
	}

	_, err = k8sClient().AppsV1().Deployments(constant.EdgeNameSpace).Get(ctx, "vk-"+nodeName, v1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		isNeedCreate = true
	}

	if !isNeedCreate {
		logrus.Infof("node %s Has Already Exist", nodeName)
		return true, nil
	}

	if err := kubeclient.CreateResourceWithFile(k8sClient(), manifests.VirtualKubeletYaml, option); err != nil {
		//rollback
		kubeclient.DeleteResourceWithFile(k8sClient(), manifests.VirtualKubeletYaml, option)
		return false, err
	}
	return false, nil
}

func deleteEdgeNode(ctx context.Context, nodeName string) error {
	option := map[string]string{
		"NodeName": nodeName,
	}
	if err := kubeclient.DeleteResourceWithFile(k8sClient(), manifests.VirtualKubeletYaml, option); err != nil {
		return err
	}
	if err := k8sClient().CoreV1().Nodes().Delete(ctx, nodeName, v1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func existEdgeNode(ctx context.Context, nodeName string) bool {
	_, err := k8sClient().CoreV1().Nodes().Get(ctx, nodeName, v1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}
