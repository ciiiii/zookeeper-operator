package constants

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	zkv1alpha1 "github.com/ciiiii/zookeeper-operator/pkg/apis/v1alpha1"
)

const (
	ClusterController = "zookeeper-controller"
	BackupController  = "backup-controller"
	RestoreController = "restore-controller"

	BinaryDir          = "/usr/local/zookeeper/bin"
	DataDir            = "/data"
	RawConfigDir       = "/conf"
	PVCVolumeName      = "data"
	ConfigVolumeName   = "config"
	BinaryVolumeName   = "bin-dir"
	ClientPort         = 2181
	FollowerPort       = 2888
	LeaderElectionPort = 3888

	ZooConfigName        = "zoo.cfg"
	ZooDynamicConfigName = "zoo.cfg.dynamic"
	Log4jCfgName         = "log4j.properties"
	Log4jQuietCfgName    = "log4j-quiet.properties"

	DOMAIN_ENV               = "DOMAIN"
	CLIENT_PORT_ENV          = "CLIENT_PORT"
	FOLLOWER_PORT_ENV        = "FOLLOWER_PORT"
	LEADER_ELECTION_PORT_ENV = "LEADER_ELECTION_PORT"
	DATA_DIR_ENV             = "DATA_DIR"
	CONFIG_DIR_ENV           = "CFG_DIR"
	RAW_CONFIG_DIR_ENV       = "RAW_CFG_DIR"
	NAMESPACE_ENV            = "NAMESPACE"

	ACCESS_KEY_ENV = "ACCESS_KEY"
	SECRET_KEY_ENV = "SECRET_KEY"
	BUCKET_ENV     = "BUCKET"
	ENDPOINT_ENV   = "ENDPOINT"
	OBJECT_KEY_ENV = "OBJECT_KEY"

	BACKUP_MOUNT_ENV  = "BACKUP_MOUNT"
	RESTORE_MOUNT_ENV = "RESTORE_MOUNT"

	BackupPrefix  = "backup-%s"
	RestorePrefix = "restore-%s"

	RolloutRestartAnnotation = "kubectl.kubrnetes.io/restartedAt"
	RestartTimeFormat        = "2006-01-02T15:04:05-0700"
)

var (
	RemovalPodLabel         = fmt.Sprintf("%s/remove", zkv1alpha1.APIGroup)
	ClearFinalizer          = fmt.Sprintf("%s/clear", zkv1alpha1.APIGroup)
	ServerIdAnnotation      = fmt.Sprintf("%s/id", zkv1alpha1.APIGroup)
	ServerReadyAnnotation   = fmt.Sprintf("%s/ready", zkv1alpha1.APIGroup)
	ServerModeAnnotation    = fmt.Sprintf("%s/mode", zkv1alpha1.APIGroup)
	ServerMessageAnnotation = fmt.Sprintf("%s/message", zkv1alpha1.APIGroup)

	BackoffLimit int32 = 0
	Completions  int32 = 1
	Parallelism  int32 = 1

	ConfigDir    = fmt.Sprintf("%s/conf", DataDir)
	HelperBinary = fmt.Sprintf("%s/zk-helper", BinaryDir)
	Size10Gi     = resource.NewQuantity(10*1024*1024*1024, resource.BinarySI)
)
