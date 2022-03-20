package helper

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/ciiiii/zookeeper-operator/pkg/client/zookeeper"
	"github.com/ciiiii/zookeeper-operator/pkg/constants"
)

func stopAction() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	<-exit
}

func removeAction(myId int, host string) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		klog.Fatalf("failed to build config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("failed to build clientset: %v", err)
	}
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), host, metav1.GetOptions{})
	if err != nil {
		klog.Fatalf("failed to get pod: %v", err)
	}
	if val := pod.Labels[constants.RemovalPodLabel]; val != "" {
		if err := os.Remove(myIdFile); err != nil {
			klog.Warningf("remove myid file failed: %v", myIdFile, err)
		} else {
			klog.Infof("remove myid file %q succeed", myIdFile)
		}
		if err := os.RemoveAll(configDir); err != nil {
			klog.Warningf("remove config dir failed: %v", err)
		} else {
			klog.Infof("remove config dir %q succeed", configDir)
		}

		zk, err := zookeeper.NewZKClient(domain, clientPort)
		if err != nil {
			klog.Fatalf("failed to connect to zk: %v", err)
		}
		if err := zk.RemoveServer(myId); err != nil {
			klog.Fatalf("failed to remove node: %v", err)
		}
		klog.Infof("node %d removed", myId)
		return
	}
	klog.Infof("no label %s, skip removing node %d", constants.RemovalPodLabel, myId)
}
