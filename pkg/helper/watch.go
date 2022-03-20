package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-zookeeper/zk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/ciiiii/zookeeper-operator/pkg/client/zookeeper"
	"github.com/ciiiii/zookeeper-operator/pkg/constants"
)

func watchAction(myId int, host string) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		klog.Fatalf("failed to build config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("failed to build clientset: %v", err)
	}
	closeCh := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go updateServerStatusTicker(myId, host, clientset, closeCh)
	for sig := range sigs {
		close(closeCh)
		waitConnectionsClose(10*time.Second, 1*time.Second)
		klog.Infof("received signal %s, handle removing node", sig)
		removeAction(myId, host)
		close(sigs)
	}
}

func updateServerStatusTicker(myId int, host string, clientset kubernetes.Interface, closeCh <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case c := <-ticker.C:
			klog.Infof("update server status at %s", c.String())
			ok, message, mode := getServerStatus()
			patch := []patchMapValue{
				{
					Op:   "replace",
					Path: "/metadata/annotations",
					Value: map[string]string{
						constants.ServerIdAnnotation:      fmt.Sprint(myId),
						constants.ServerReadyAnnotation:   fmt.Sprint(ok),
						constants.ServerMessageAnnotation: message,
						constants.ServerModeAnnotation:    mode,
					},
				},
			}
			patchBytes, err := json.Marshal(patch)
			if err != nil {
				klog.Warningf("failed to encode patch: %v", err)
				continue
			}
			if _, err := clientset.CoreV1().Pods(namespace).Patch(context.TODO(), host, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
				klog.Warningf("failed to patch pod: %v", err)
			}
		case <-closeCh:
			klog.Info("stop update server status")
			return
		}
	}
}

func getServerStatus() (ok bool, message string, mode string) {
	ok = zookeeper.Ruok("localhost", clientPort)
	stat, err := zookeeper.Srvr("localhost", clientPort)
	if err != nil {
		message = fmt.Sprintf("failed to get server status: %v", err)
		mode = zk.ModeUnknown.String()
	} else {
		message = ""
		mode = stat.Mode.String()
	}
	return
}

type patchMapValue struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value map[string]string `json:"value"`
}

func waitConnectionsClose(timeout, period time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(period)
	for {
		select {
		case <-ticker.C:
			count, err := countRemoteConnections()
			if err != nil {
				klog.Warningf("failed to check connections: %v", err)
				continue
			}
			if count == 0 {
				return
			} else {
				klog.Infof("waiting connections close: %d", count)
			}
		case <-timer.C:
			klog.Infof("timeout(%s) to wait connections close", timeout.String())
			return
		}
	}
}

func countRemoteConnections() (int, error) {
	conns, err := zookeeper.Cons("localhost", clientPort)
	if err != nil {
		return -1, err
	}
	if len(conns) == 0 {
		return 0, nil
	}
	count := 0
	for _, conn := range conns {
		if !strings.Contains(conn.Addr, "127.0.0.1") {
			count++
		}
	}
	return count, nil
}
