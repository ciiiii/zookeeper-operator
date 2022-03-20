package helper

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"k8s.io/klog"

	"github.com/ciiiii/zookeeper-operator/pkg/client/zookeeper"
)

func readyAction(myId int, host string) {
	// check server with local connection
	if ok := zookeeper.Ruok("localhost", clientPort); !ok {
		klog.Fatal("zk is not ok")
	}

	localClient, err := zookeeper.NewZKClient("localhost", clientPort)
	if err != nil {
		klog.Fatalf("failed to create local zk client: %v", err)
	}
	localClient.Close()
	// check server with headless service connection
	headlessClient, err := zookeeper.NewZKClient(fmt.Sprintf("%s.%s", host, domain), clientPort)
	if err != nil {
		klog.Fatalf("failed to create remote zk client: %v", err)
	}
	headlessClient.Close()

	// get role of current server
	f, err := ioutil.ReadFile(dynamicConfig)
	if err != nil {
		klog.Fatalf("failed to read dynamic config file: %v", err)
	}
	servers := strings.Split(string(f), "\n")
	currentServer, err := parseCurrentServer(myId, servers)
	if err != nil {
		klog.Fatalf("failed to parse current server: %v", err)
	}
	currentRole := parseServerRole(currentServer)
	switch currentRole {
	case "observer":
		klog.Infof("role of current server is observer, try to update")
		remoteClient, err := zookeeper.NewZKClient(domain, clientPort)
		if err != nil {
			klog.Fatalf("failed to create remote zk client: %v", err)
		}
		defer remoteClient.Close()
		// ref: [Additional comments]#Changing an observer into a follower
		if err := remoteClient.RemoveServer(myId); err != nil {
			klog.Fatalf("failed to remove server(%d) via %s:2181: %v", myId, domain, err)
		}

		// register node with backoff retry
		backoff.RetryNotify(
			func() error {
				return remoteClient.RegisterServer(myId, generateServerStr(host, ROLE_PARTICIPANT))
			},
			backoff.NewExponentialBackOff(),
			func(err error, d time.Duration) {
				klog.Infof("retry after %s: %v", d, err)
			})

		servers, err := remoteClient.ListServer()
		if err != nil {
			klog.Fatalf("failed to list server: %v", err)
		}
		fmt.Println("servers: ", strings.Join(servers, ", "))

		if err := forceWriteFile(dynamicConfig, []byte(strings.Join(servers, "\n"))); err != nil {
			klog.Fatalf("failed to write dynamic config file: %v", err)
		}
	case "participant":
		klog.Infof("role of current server is participant")
	default:
		klog.Fatalf("failed to parse current server role: %s", currentServer)
	}
}
