package helper

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/klog/v2"

	"github.com/ciiiii/zookeeper-operator/pkg/constants"
)

var (
	namespace          = os.Getenv(constants.NAMESPACE_ENV)
	domain             = os.Getenv(constants.DOMAIN_ENV)
	rawConfigDir       = envWithDefault(constants.RAW_CONFIG_DIR_ENV, constants.RawConfigDir)
	dataDir            = envWithDefault(constants.DATA_DIR_ENV, constants.DataDir)
	clientPort         = envWithDefault(constants.CLIENT_PORT_ENV, fmt.Sprint(constants.ClientPort))
	followerPort       = envWithDefault(constants.FOLLOWER_PORT_ENV, fmt.Sprint(constants.FollowerPort))
	leaderElectionPort = envWithDefault(constants.LEADER_ELECTION_PORT_ENV, fmt.Sprint(constants.LeaderElectionPort))

	bucket     = os.Getenv(constants.BUCKET_ENV)
	endpoint   = os.Getenv(constants.ENDPOINT_ENV)
	objectKey  = os.Getenv(constants.OBJECT_KEY_ENV)
	accessKey  = os.Getenv(constants.ACCESS_KEY_ENV)
	secretKey  = os.Getenv(constants.SECRET_KEY_ENV)
	backupDir  = os.Getenv(constants.BACKUP_MOUNT_ENV)
	restoreDir = os.Getenv(constants.RESTORE_MOUNT_ENV)

	rawStaticConfig = fmt.Sprintf("%s/%s", rawConfigDir, constants.ZooConfigName)
	configDir       = fmt.Sprintf("%s/conf", dataDir)
	staticConfig    = fmt.Sprintf("%s/%s", constants.ConfigDir, constants.ZooConfigName)
	dynamicConfig   = fmt.Sprintf("%s/%s", dataDir, constants.ZooDynamicConfigName)
	myIdFile        = fmt.Sprintf("%s/myid", dataDir)

	action     string
	kubeConfig string
)

const (
	ROLE_OBSERVER    = "observer"
	ROLE_PARTICIPANT = "participant"
	ROLE_UNKNOWN     = "unknown"

	ACTION_INIT    = "init"
	ACTION_READY   = "ready"
	ACTION_WATCH   = "watch"
	ACTION_STOP    = "stop"
	ACTION_BACKUP  = "backup"
	ACTION_RESTORE = "restore"
)

func init() {
	flag.StringVar(&action, "action", "init", "action to perform")
	flag.StringVar(&kubeConfig, "kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig file")
	klog.InitFlags(nil)
	flag.Parse()
}

func Run() {
	host, err := os.Hostname()
	if err != nil {
		klog.Fatalf("failed to get hostname: %v", err)
	}
	switch action {
	case ACTION_INIT:
		initAction(parseMyId(host), host)
	case ACTION_READY:
		readyAction(parseMyId(host), host)
	case ACTION_WATCH:
		watchAction(parseMyId(host), host)
	case ACTION_STOP:
		stopAction()
	case ACTION_BACKUP:
		backupAction(host)
	case ACTION_RESTORE:
		restoreAction(host)
	default:
		klog.Fatalf("unknown action: %s", action)
	}
}
