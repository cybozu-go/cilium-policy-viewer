package cmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/cilium/cilium/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	directionEgress  = "EGRESS"
	directionIngress = "INGRESS"
)

func createClients(ctx context.Context, name string) (*kubernetes.Clientset, *dynamic.DynamicClient, *client.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	ciliumClient, err := createCiliumClient(ctx, clientset, rootOptions.namespace, name)
	if err != nil {
		return nil, nil, nil, err
	}

	return clientset, dynamicClient, ciliumClient, err
}

func createCiliumClient(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*client.Client, error) {
	endpoint, err := getProxyEndpoint(ctx, clientset, namespace, name)
	if err != nil {
		return nil, err
	}
	client, err := client.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getProxyEndpoint(ctx context.Context, c *kubernetes.Clientset, namespace, name string) (string, error) {
	targetPod, err := c.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	targetNode := targetPod.Spec.NodeName

	pods, err := c.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + targetNode,
		LabelSelector: rootOptions.proxySelector,
	})
	if err != nil {
		return "", err
	}
	if num := len(pods.Items); num != 1 {
		err := fmt.Errorf("failed to find cilium-agent-proxy. found %d pods", num)
		return "", err
	}

	podIP := pods.Items[0].Status.PodIP
	return fmt.Sprintf("http://%s:%d", podIP, rootOptions.proxyPort), nil
}

func getPodEndpointID(ctx context.Context, d *dynamic.DynamicClient, namespace, name string) (int64, int64, error) {
	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumendpoints",
	}

	ep, err := d.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return 0, 0, err
	}

	endpointID, found, err := unstructured.NestedInt64(ep.Object, "status", "id")
	if err != nil {
		return 0, 0, err
	}
	if !found {
		return 0, 0, errors.New("CiliumEndpoint does not have .status.id")
	}

	endpointIdentity, found, err := unstructured.NestedInt64(ep.Object, "status", "identity", "id")
	if err != nil {
		return 0, 0, err
	}
	if !found {
		return 0, 0, errors.New("CiliumEndpoint does not have .status.identity.id")
	}

	return endpointID, endpointIdentity, nil
}

func listCiliumIDs(ctx context.Context, d *dynamic.DynamicClient) (*unstructured.UnstructuredList, error) {
	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumidentities",
	}
	return d.Resource(gvr).List(ctx, metav1.ListOptions{})
}

func findCiliumID(dict *unstructured.UnstructuredList, id int64) *unstructured.Unstructured {
	if dict == nil {
		return nil
	}
	name := strconv.FormatInt(id, 10)
	for _, item := range dict.Items {
		if item.GetName() == name {
			return &item
		}
	}
	return nil
}
