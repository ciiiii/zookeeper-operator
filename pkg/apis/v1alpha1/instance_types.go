package v1alpha1

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ZooKeeperInstance struct {
	// +optional
	LabelSelector map[string]string `json:"labelSelector,omitempty"`
	// +optional
	Name string `json:"name,omitempty"`
	Host string `json:"host,omitempty"`
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`
	// +optional
	// +kubebuilder:default="/data/version-2"
	DataDir string `json:"dataDir,omitempty"`
}

func (instance *ZooKeeperInstance) GetStatefulSet(cacheClient client.Client, namespace string) (*appsv1.StatefulSet, string, error) {
	sts := &appsv1.StatefulSet{}
	switch {
	case instance.Name != "":
		err := cacheClient.Get(context.TODO(), client.ObjectKey{
			Namespace: namespace,
			Name:      instance.Name,
		}, sts)
		switch {
		case err != nil && errors.IsNotFound(err):
			return nil, fmt.Sprintf("no statefulset name %s found", instance.Name), nil
		case err != nil:
			return nil, "", err
		}
	case instance.LabelSelector != nil:
		stss := &appsv1.StatefulSetList{}
		labelSelector := client.MatchingLabels(instance.LabelSelector)
		err := cacheClient.List(context.TODO(), stss, client.InNamespace(namespace), labelSelector)
		switch {
		case err != nil:
			return nil, "", err
		case len(stss.Items) == 0:
			return nil, fmt.Sprintf("no statefulset match with %s found", instance.LabelSelector), nil
		case len(stss.Items) > 1:
			return nil, fmt.Sprintf("more than one statefulset match with %s found", instance.LabelSelector), nil
		default:
			sts = &stss.Items[0]
		}
	default:
		return nil, "", fmt.Errorf("source.name or source.labelSelector is required")
	}
	return sts, "", nil
}
