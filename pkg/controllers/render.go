package controllers

import (
	"fmt"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	zkv1alpha1 "github.com/ciiiii/zookeeper-operator/pkg/apis/v1alpha1"
	"github.com/ciiiii/zookeeper-operator/pkg/constants"
)

func renderStatefulset(cluster *zkv1alpha1.ZooKeeperCluster) *appsv1.StatefulSet {
	domain := fmt.Sprintf("%s.%s.svc.%s", cluster.Name, cluster.Namespace, cluster.Spec.ClusterDomain)
	cmName := fmt.Sprintf("%s-config", cluster.Name)
	saName := fmt.Sprintf("%s-sa", cluster.Name)

	env := []corev1.EnvVar{
		{Name: constants.DOMAIN_ENV, Value: domain},
		{Name: constants.DATA_DIR_ENV, Value: cluster.Spec.Config.DataDir},
		{Name: constants.CONFIG_DIR_ENV, Value: cluster.Spec.Config.ConfigDir},
		{Name: constants.RAW_CONFIG_DIR_ENV, Value: cluster.Spec.Config.RawConfigDir},
		{Name: constants.CLIENT_PORT_ENV, Value: fmt.Sprint(cluster.Spec.Config.ClientPort)},
		{Name: constants.FOLLOWER_PORT_ENV, Value: fmt.Sprint(cluster.Spec.Config.FollowerPort)},
		{Name: constants.LEADER_ELECTION_PORT_ENV, Value: fmt.Sprint(cluster.Spec.Config.LeaderElectionPort)},
		{Name: constants.NAMESPACE_ENV, ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
	}

	dataMount := corev1.VolumeMount{Name: constants.PVCVolumeName, MountPath: cluster.Spec.Config.DataDir}
	configMount := corev1.VolumeMount{Name: constants.ConfigVolumeName, MountPath: cluster.Spec.Config.RawConfigDir}
	binaryMount := corev1.VolumeMount{Name: constants.BinaryVolumeName, MountPath: constants.BinaryDir}

	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.PVCVolumeName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},

			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *constants.Size10Gi,
				},
			},
		},
	}

	binaryEmptyDir := corev1.Volume{
		Name: constants.BinaryVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	configCM := corev1.Volume{
		Name: constants.ConfigVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
			},
		},
	}

	sts := &appsv1.StatefulSet{}
	sts.Name = cluster.Name
	sts.Namespace = cluster.Namespace
	sts.Spec.Replicas = &cluster.Spec.Replicas
	sts.Spec.ServiceName = cluster.Name
	sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{pvc}
	initContainer := corev1.Container{
		Name:  "init",
		Image: cluster.Spec.HelperImage,
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf(initCommand, constants.HelperBinary, cluster.Spec.Config.RawConfigDir, cluster.Spec.Config.ConfigDir, constants.Log4jCfgName, constants.Log4jQuietCfgName),
		},
		Env: env,
		VolumeMounts: []corev1.VolumeMount{
			dataMount,
			configMount,
			binaryMount,
		},
	}
	zookeeperContainer := corev1.Container{
		Name:  "zookeeper",
		Image: cluster.Spec.Image,
		Command: []string{
			"zkServer.sh",
			"--config",
			cluster.Spec.Config.ConfigDir,
			"start-foreground",
		},
		Ports: []corev1.ContainerPort{
			{ContainerPort: cluster.Spec.Config.ClientPort},
			{ContainerPort: cluster.Spec.Config.FollowerPort},
			{ContainerPort: cluster.Spec.Config.LeaderElectionPort},
		},
		Env: env,
		VolumeMounts: []corev1.VolumeMount{
			dataMount,
			binaryMount,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						constants.HelperBinary,
						"--action=ready",
					},
				},
			},
			FailureThreshold: 3,
			PeriodSeconds:    10,
			SuccessThreshold: 1,
			TimeoutSeconds:   5,
		},
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						constants.HelperBinary,
						"--action=stop",
					},
				},
			},
		},
	}
	helperContainer := corev1.Container{
		Name:  "zookeeper-helper",
		Image: cluster.Spec.HelperImage,
		Command: []string{
			"/usr/local/bin/zk-helper",
			"--action=watch",
		},
		Env:          env,
		VolumeMounts: []corev1.VolumeMount{dataMount},
	}
	selector := map[string]string{
		"operator": "zookeeper",
		"instance": cluster.Name,
	}
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: selector,
	}
	sts.Spec.Template.Labels = selector
	sts.Spec.Template.Spec.ServiceAccountName = saName
	sts.Spec.Template.Spec.InitContainers = []corev1.Container{initContainer}
	sts.Spec.Template.Spec.Containers = []corev1.Container{zookeeperContainer, helperContainer}
	sts.Spec.Template.Spec.Volumes = []corev1.Volume{binaryEmptyDir, configCM}
	return sts
}

func renderConigMap(cluster *zkv1alpha1.ZooKeeperCluster) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.Name = fmt.Sprintf("%s-config", cluster.Name)
	cm.Namespace = cluster.Namespace
	cm.Data = map[string]string{
		cluster.Spec.Config.StaticConfig: fmt.Sprintf(zooCfg, cluster.Spec.Config.DataDir, cluster.Spec.Config.DynamicConfig),
		constants.Log4jCfgName:           log4jCfg,
		constants.Log4jQuietCfgName:      log4jQuietCfg,
	}
	return cm
}

func renderRBAC(cluster *zkv1alpha1.ZooKeeperCluster) []runtime.Object {
	roleName := fmt.Sprintf("%s-role", cluster.Name)
	serviceAccountName := fmt.Sprintf("%s-sa", cluster.Name)

	serviceAccount := &corev1.ServiceAccount{}
	serviceAccount.Name = serviceAccountName
	serviceAccount.Namespace = cluster.Namespace

	role := &rbacv1.Role{}
	role.Name = roleName
	role.Namespace = cluster.Namespace
	role.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"*"},
		},
	}

	roleBinding := &rbacv1.RoleBinding{}
	roleBinding.Name = fmt.Sprintf("%s-rolebinding", cluster.Name)
	roleBinding.Namespace = cluster.Namespace
	roleBinding.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     roleName,
	}
	roleBinding.Subjects = []rbacv1.Subject{
		{
			APIGroup: corev1.GroupName,
			Kind:     "ServiceAccount",
			Name:     serviceAccountName,
		},
	}
	return []runtime.Object{serviceAccount, role, roleBinding}
}

func renderService(cluster *zkv1alpha1.ZooKeeperCluster) *corev1.Service {
	service := &corev1.Service{}
	service.Name = cluster.Name
	service.Namespace = cluster.Namespace
	service.Spec.Type = corev1.ServiceTypeClusterIP
	service.Spec.ClusterIP = corev1.ClusterIPNone
	service.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "client",
			Port:       cluster.Spec.Config.ClientPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(int(cluster.Spec.Config.ClientPort)),
		},
		{
			Name:       "follower",
			Port:       cluster.Spec.Config.FollowerPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(int(cluster.Spec.Config.FollowerPort)),
		},
		{
			Name:       "leader-election",
			Port:       cluster.Spec.Config.LeaderElectionPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(int(cluster.Spec.Config.LeaderElectionPort)),
		},
	}
	service.Spec.Selector = map[string]string{
		"operator": "zookeeper",
		"instance": cluster.Name,
	}
	service.Spec.SessionAffinity = corev1.ServiceAffinityNone
	return service
}

func renderJob(backup *zkv1alpha1.ZooKeeperBackup, pvcName string, stroageLocation *zkv1alpha1.GenericStroageLocation) *batchv1.Job {
	job := &batchv1.Job{}
	job.Name = fmt.Sprintf(constants.BackupPrefix, backup.Name)
	job.Namespace = backup.Namespace
	job.Labels = map[string]string{
		"operator": "zookeeper-backup",
		"instance": backup.Name,
	}
	job.Spec.Template = renderJobPodTemplate(backup, pvcName, stroageLocation)
	job.Spec.BackoffLimit = &constants.BackoffLimit
	job.Spec.Completions = &constants.Completions
	job.Spec.Parallelism = &constants.Parallelism

	return job
}

func renderCronJob(backup *zkv1alpha1.ZooKeeperBackup, pvcName string, stroageLocation *zkv1alpha1.GenericStroageLocation) *batchv1beta1.CronJob {
	job := &batchv1beta1.CronJob{}
	job.Name = fmt.Sprintf(constants.BackupPrefix, backup.Name)
	job.Namespace = backup.Namespace
	job.Labels = map[string]string{
		"operator": "zookeeper-backup",
		"instance": backup.Name,
	}
	job.Spec.Schedule = backup.Spec.Schedule
	job.Spec.Suspend = &backup.Spec.Suspend
	job.Spec.JobTemplate.Labels = map[string]string{
		"mode":     "schedule-backup",
		"template": job.Name,
	}
	job.Spec.JobTemplate.Spec.Template = renderJobPodTemplate(backup, pvcName, stroageLocation)
	job.Spec.JobTemplate.Spec.BackoffLimit = &constants.BackoffLimit
	job.Spec.JobTemplate.Spec.Completions = &constants.Completions
	job.Spec.JobTemplate.Spec.Parallelism = &constants.Parallelism
	return job
}

func renderJobPodTemplate(backup *zkv1alpha1.ZooKeeperBackup, pvcName string, stroageLocation *zkv1alpha1.GenericStroageLocation) corev1.PodTemplateSpec {
	pod := corev1.PodTemplateSpec{}
	pod.Labels = map[string]string{
		"operator": "zookeeper-backup",
		"instance": backup.Name,
	}
	pvcVolumeName := "data-pvc"
	pvcMountPath := "/backup"
	pvcVolume := corev1.Volume{
		Name: pvcVolumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	}

	pvcMount := corev1.VolumeMount{
		Name:      pvcVolumeName,
		MountPath: pvcMountPath,
	}

	env := []corev1.EnvVar{
		{Name: constants.BUCKET_ENV, Value: stroageLocation.Bucket},
		{Name: constants.ENDPOINT_ENV, Value: stroageLocation.Endpoint},
		{Name: constants.OBJECT_KEY_ENV, Value: stroageLocation.Key},
		{Name: constants.DATA_DIR_ENV, Value: backup.Spec.Source.DataDir},
		{Name: constants.BACKUP_MOUNT_ENV, Value: pvcMountPath},
		{
			Name: constants.ACCESS_KEY_ENV,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: stroageLocation.AccessKey,
			},
		},
		{
			Name: constants.SECRET_KEY_ENV,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: stroageLocation.SecretKey,
			},
		},
		{
			Name: constants.NAMESPACE_ENV,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	container := corev1.Container{
		Name:  "zookeeper-backup",
		Image: backup.Spec.Image,
		Command: []string{
			"/usr/local/bin/zk-helper",
			"--action=backup",
		},
		Env:          env,
		VolumeMounts: []corev1.VolumeMount{pvcMount},
	}
	pod.Spec.Containers = []corev1.Container{container}
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.Volumes = []corev1.Volume{pvcVolume}
	return pod
}

func renderRestoreJob(restore *zkv1alpha1.ZooKeeperRestore, pvcList []string, stroageLocation *zkv1alpha1.GenericStroageLocation) *batchv1.Job {
	job := &batchv1.Job{}
	job.Name = fmt.Sprintf(constants.RestorePrefix, restore.Name)
	job.Namespace = restore.Namespace
	job.Labels = map[string]string{
		"operator": "zookeeper-restore",
		"instance": restore.Name,
	}
	job.Spec.Template = renderRestoreJobPodTemplate(restore, pvcList, stroageLocation)
	job.Spec.BackoffLimit = &constants.BackoffLimit
	job.Spec.Completions = &constants.Completions
	job.Spec.Parallelism = &constants.Parallelism

	return job
}

func renderRestoreJobPodTemplate(restore *zkv1alpha1.ZooKeeperRestore, pvcList []string, stroageLocation *zkv1alpha1.GenericStroageLocation) corev1.PodTemplateSpec {
	pod := corev1.PodTemplateSpec{}
	pod.Labels = map[string]string{
		"operator": "zookeeper-restore",
		"instance": restore.Name,
	}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	mountPaths := []string{}

	for index, pvc := range pvcList {
		volumeName := fmt.Sprintf("pvc-%d", index)
		mountPath := filepath.Join("/restore", volumeName)
		mountPaths = append(mountPaths, mountPath)
		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
		})
	}

	env := []corev1.EnvVar{
		{Name: constants.BUCKET_ENV, Value: stroageLocation.Bucket},
		{Name: constants.ENDPOINT_ENV, Value: stroageLocation.Endpoint},
		{Name: constants.OBJECT_KEY_ENV, Value: stroageLocation.Key},
		{Name: constants.DATA_DIR_ENV, Value: restore.Spec.Target.DataDir},
		{Name: constants.RESTORE_MOUNT_ENV, Value: strings.Join(mountPaths, ",")},
		{
			Name: constants.ACCESS_KEY_ENV,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: stroageLocation.AccessKey,
			},
		},
		{
			Name: constants.SECRET_KEY_ENV,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: stroageLocation.SecretKey,
			},
		},
		{
			Name: constants.NAMESPACE_ENV,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	container := corev1.Container{
		Name:  "zookeeper-restore",
		Image: restore.Spec.Image,
		Command: []string{
			"/usr/local/bin/zk-helper",
			"--action=restore",
		},
		Env:          env,
		VolumeMounts: volumeMounts,
	}
	pod.Spec.Containers = []corev1.Container{container}
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.Volumes = volumes
	return pod
}
