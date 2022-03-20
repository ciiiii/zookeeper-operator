package helper

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"k8s.io/klog"

	"github.com/ciiiii/zookeeper-operator/pkg/client/zookeeper"
)

func initAction(myId int, host string) {
	if err := syncMyIdFile(myId); err != nil {
		klog.Fatalf("failed to sync myid file: %v", err)
	}
	klog.Infof("sync myid file to %d", myId)

	if err := syncDynamicConfig(host, myId); err != nil {
		klog.Fatalf("failed to sync dynamic config: %v", err)
	}
	klog.Info("sync dynamic config succeed")

	if err := syncStaticConfig(); err != nil {
		klog.Fatalf("failed to sync static config: %v", err)
	}
	klog.Info("sync static config succeed")
}

func syncMyIdFile(myId int) error {
	myIdSynced := false
	_, err := os.Stat(myIdFile)
	switch {
	case err != nil && os.IsNotExist(err):
		klog.Infof("myid file not exists, writing to %q directly", myIdFile)
	case err != nil:
		return err
	default:
		f, err := os.Open(myIdFile)
		if err != nil {
			return err
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		existingMyId, err := strconv.Atoi(string(b))
		if err != nil {
			return err
		}
		if existingMyId == myId {
			myIdSynced = true
		} else {
			klog.Warningf("myid(%d) in %s is not expected %d", existingMyId, myIdFile, myId)
		}
	}
	if !myIdSynced {
		klog.Infof("writing myid(%d) to %q", myId, myIdFile)
		if err := forceWriteFile(myIdFile, []byte(strconv.Itoa(myId))); err != nil {
			return err
		}
	}
	return nil
}

// [Adding servers]#Option1
// first server write dynamic config directly to init cluster
// second server join cluster as observer first, will switch to participant later
// left servers join cluster as participant directly
func syncDynamicConfig(host string, myId int) error {
	switch myId {
	case 1:
		if err := initCluster(host, myId); err != nil {
			return err
		}
	case 2:
		if err := joinCluster(myId, host, ROLE_OBSERVER); err != nil {
			return err
		}
	default:
		if err := joinCluster(myId, host, ROLE_PARTICIPANT); err != nil {
			return err
		}
	}
	// check if dynamic config exists last
	// if dynamic config not exists, zookeeper will crash
	_, err := os.Stat(dynamicConfig)
	if err != nil {
		return err
	}
	return nil
}

func initCluster(host string, myId int) error {
	dynamicConfigured := false
	_, err := os.Stat(dynamicConfig)
	switch {
	case err != nil && os.IsNotExist(err):
		klog.Infof("dynamic config not exists, writing to %q directly", dynamicConfig)
	case err != nil:
		return err
	default:
		dynamicConfigured = true
		klog.Infof("dynamic config exists")
	}
	if !dynamicConfigured {
		klog.Infof("writing configuration to %q", dynamicConfig)
		serverStr := fmt.Sprintf("server.%d=%s", myId, generateServerStr(host, ROLE_PARTICIPANT))
		if err := forceWriteFile(dynamicConfig, []byte(serverStr)); err != nil {
			return err
		}
	}
	return nil
}

func joinCluster(myId int, host, role string) error {
	ensembleAvailable, err := checkEnsembleAvailable(domain)
	if err != nil {
		return fmt.Errorf("failed to check ensemble availability: %v", err)
	}
	if !ensembleAvailable {
		return fmt.Errorf("ensemble is unavailable")
	}
	zk, err := zookeeper.NewZKClient(domain, clientPort)
	if err != nil {
		return err
	}
	defer zk.Close()
	exist, err := zk.ExistServer(myId)
	if err != nil {
		return err
	}
	if exist {
		klog.Infof("server %d already registered", myId)
		return nil
	}
	if err := zk.RegisterServer(myId, generateServerStr(host, role)); err != nil {
		return err
	}
	servers, err := zk.ListServer()
	if err != nil {
		return err
	}
	if err := forceWriteFile(dynamicConfig, []byte(strings.Join(servers, "\n"))); err != nil {
		return err
	}
	return nil
}

func syncStaticConfig() error {
	rawConfigBytes, err := ioutil.ReadFile(rawStaticConfig)
	if err != nil {
		return err
	}
	_, statErr := os.Stat(staticConfig)
	switch {
	case statErr != nil && os.IsNotExist(statErr):
		klog.Infof("static config not exists, writing to %q directly", staticConfig)
		if err := forceWriteFile(staticConfig, rawConfigBytes); err != nil {
			return err
		}
		return nil
	case statErr != nil:
		return statErr
	default:
		klog.Infof("static config exists, merge configs")
	}
	// TODO: choose which dynamic config should to use
	// register node => use generated dynamic config
	// else => use previous dynamic config
	configPrefix := "dynamicConfigFile="
	configBytes, err := ioutil.ReadFile(staticConfig)
	if err != nil {
		return err
	}
	existDynmaicConfig := parseSpecificLine(configBytes, configPrefix)
	if existDynmaicConfig != "" {
		configLines := append(dropSpecificLine(rawConfigBytes, configPrefix), existDynmaicConfig)
		if err := forceWriteFile(staticConfig, []byte(strings.Join(configLines, "\n"))); err != nil {
			return err
		}
	} else {
		if err := forceWriteFile(staticConfig, rawConfigBytes); err != nil {
			return err
		}
	}
	return nil
}

func dropSpecificLine(b []byte, prefix string) []string {
	var lines []string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, prefix) {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func parseSpecificLine(b []byte, prefix string) string {
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func oldInitAction(myId int, host string) {
	// check if myid file exists and correct
	myIdConfigured := false
	_, myIdFileStatErr := os.Stat(myIdFile)
	switch {
	case myIdFileStatErr != nil && os.IsNotExist(myIdFileStatErr):

	case myIdFileStatErr != nil:
		klog.Fatalf("failed to stat myid file: %v", myIdFileStatErr)
	default:
		f, err := os.Open(myIdFile)
		if err != nil {
			klog.Fatalf("failed to open myid file: %v", err)
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			klog.Fatalf("failed to read myid file: %v", err)
		}
		existingMyId, err := strconv.Atoi(string(b))
		if err != nil {
			klog.Fatalf("failed to parse myid %s: %v", b, err)
		}
		if existingMyId == myId {
			myIdConfigured = true
		} else {
			klog.Warningf("myid(%d) in %s is not expected %d", existingMyId, myIdFile, myId)
		}
	}

	// check if dynamic config exists
	dynamicConfigured := false
	_, dynamicConfigStatErr := os.Stat(dynamicConfig)
	switch {
	case dynamicConfigStatErr != nil && os.IsNotExist(dynamicConfigStatErr):
	case dynamicConfigStatErr != nil:
		klog.Fatalf("failed to stat dynamic config file: %v", dynamicConfigStatErr)
	default:
		dynamicConfigured = true
	}

	// write config when myId and dynamic config both are not configured
	if !myIdConfigured || !dynamicConfigured {
		klog.Infof("writing myid(%d) to %q", myId, myIdFile)
		if err := forceWriteFile(myIdFile, []byte(strconv.Itoa(myId))); err != nil {
			klog.Fatalf("failed to write myid file: %v", err)
		}
		// first node write dynamic config directly
		if myId == 1 {
			klog.Infof("writing configuration to %q", dynamicConfig)
			serverStr := fmt.Sprintf("server.%d=%s", myId, generateServerStr(host, ROLE_PARTICIPANT))
			if err := forceWriteFile(dynamicConfig, []byte(serverStr)); err != nil {
				klog.Fatalf("failed to write dynamic config file: %v", err)
			}
		}
	}

	// check if ensemble is available
	ensembleAvailable, err := checkEnsembleAvailable(domain)
	if err != nil {
		klog.Fatalf("failed to check ensemble availability: %v", err)
	}

	if myIdConfigured {
		klog.Infof("myIdFile %s has been configured", myIdFile)
	} else {
		klog.Infof("myIdFile %s hasn't been configured", myIdFile)
	}
	if dynamicConfigured {
		klog.Infof("dynamicConfig %s has been configured", dynamicConfig)
	} else {
		klog.Infof("dynamicConfig %s hasn't been configured", dynamicConfig)
	}
	if ensembleAvailable {
		klog.Infof("ensemble is available")
	} else {
		klog.Infof("ensemble is unavailable")
	}

	// determine register node or not
	registerNode := true
	switch {
	case !ensembleAvailable:
		registerNode = false
	case !myIdConfigured || !dynamicConfigured:
		registerNode = true
	default:
		registerNode = false
	}

	if registerNode {
		klog.Infof("registering node %d", myId)
		zk, err := zookeeper.NewZKClient(domain, clientPort)
		if err != nil {
			klog.Fatal(err)
		}
		defer zk.Close()
		// [Adding servers]#Option1
		// second server join as observer first, will switch to participant later in readinessProbe
		// other server join as participant directly
		if myId == 2 {
			if err := zk.RegisterServer(myId, generateServerStr(host, ROLE_OBSERVER)); err != nil {
				klog.Fatalf("failed to register server: %v", err)
			}
		} else {
			if err := zk.RegisterServer(myId, generateServerStr(host, ROLE_PARTICIPANT)); err != nil {
				klog.Fatalf("failed to register server: %v", err)
			}
		}

		servers, err := zk.ListServer()
		if err != nil {
			klog.Fatalf("failed to list server: %v", err)
		}
		klog.Infof("servers: %s", strings.Join(servers, ", "))
		f, err := os.Create(dynamicConfig)
		if err != nil {
			klog.Fatalf("failed to create dynamic config file: %v", err)
		}
		defer f.Close()
		if err := forceWriteFile(dynamicConfig, []byte(strings.Join(servers, "\n"))); err != nil {
			klog.Fatalf("failed to write dynamic config file: %v", err)
		}
	}

	// copy /conf(from ConfigMap:ro) dir to /data/conf(from PVC:rw)

	// check if dynamic config exists last
	// if dynamic config not exists, zookeeper will crash
	if _, err := os.Stat(dynamicConfig); os.IsNotExist(err) {
		klog.Fatalf("dynamic config file %q is not found", dynamicConfig)

	} else if err != nil {
		klog.Fatalf("failed to stat dynamic config file: %v", err)
	}
}
