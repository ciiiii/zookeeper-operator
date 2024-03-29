# zookeeper-operator
[![Actions Status](https://github.com/ciiiii/zookeeper-operator/workflows/Publish%20Docker%20image/badge.svg)](https://github.com/ciiiii/zookeeper-operator/actions)

## CRD
### ZooKeeperCluster
```yaml
apiVersion: zookeeper.example.com/v1alpha1
kind: ZooKeeperCluster
metadata:
  name: test
  namespace: zookeeper
spec:
  replicas: 3
  clusterDomain: cluster.local
  clearPersistence: true
  image: zookeeper:3.7.0
  helperImage: go2sheep/zk-helper:latest
  config:
    clientPort: 2181
    followerPort: 2888
    leaderElectionPort: 3888
    dataDir: /data
    rawConfigDir: /conf
    configDir: /data/conf
    staticConfig: zoo.cfg
    dynamicConfig: zoo.cfg.dynamic
```
#### Specification
**NOTICE**: the older images have some problems, if you have pulled, please use `helperImage: go2sheep/zk-helper:10cc9d8` to pull new one.
- replicas: number of zookeeper servers, can be dynamicly modified
- cluterDomain: must be set correctly with 
- clearPersistence: clear PVC when cluster destory or not
- image: zookeeper image
- helperImage: zookeeper helper image, used to init server, report, clear data
- config: zookeeper config options, all have default values
#### Status
```yaml
status:
  readyReplicas: 3
  replicas: 3
  servers:
  - mode: leader
    myId: "1"
    name: test-0
    ready: "true"
  - mode: follower
    myId: "2"
    name: test-1
    ready: "true"
  - mode: follower
    myId: "3"
    name: test-2
    ready: "true"
  service: test.zookeeper.svc.cluster.local
```
- readyReplicas: number of ready zookeeper servers
- replicas: number of desired zookeeper servers
- servers: list of zookeeper servers
    - mode: leader, follower
    - myId: server id
    - name: server Pod name
    - ready: server ready or not
    - message: error message
- service: zookeeper service host
- conditions: to be implemented

### ZooKeeperBackup
```yaml
apiVersion: zookeeper.example.com/v1alpha1
kind: ZooKeeperBackup
metadata:
  name: test
  namespace: zookeeper
spec:
  image: go2sheep/zk-helper:latest
  mode: once
  source:
    name: test
    dataDir: /version-2
    host: zk-admin.zookeeper
    port: 8080
  target:
    oss:
      endpoint: "oss-cn-shanghai.aliyuncs.com"
      bucket: "zookeeper-test"
      accessKeySecret:
        name: oss-key
        key: accessKey
      secretKeySecret:
        name: oss-key
        key: secretKey
```
#### Specification
**NOTICE**: the older images have some problems, if you have pulled, please use `image: go2sheep/zk-helper:10cc9d8` to pull new one.
- image: zookeeper backup job image
- mode: backup mode, can be once, shedule
- schedule: backup schedule, only valid when mode is shedule, format is cron
- suspend: used to suspend job in shedule mode
- source: backup source, must be set
    - can use name or label selector to specify source statefulset
    - host and port is used to call admin api
    - dataDir is used to specify zookeeper data directory
- target: backup target, must be set, can use oss or s3
    - oss is alicloud oss, must set endpoint, bucket, accessKeySecret and secretKeySecret
    - s3 is not supported yet
#### Status
```yaml
status:
  record:
  - key: zookeeper/test/once/backup-test-jw7dc
    startTime: "2022-03-21T01:26:36Z"
    finishTime: "2022-03-21T01:26:47Z"
  status: completed
```
- status: backup status, can be completed, failed, running, pending
- message: backup error message
- record: backup record, list of backup jobs
    - key: objectKey of backup directory in oss or s3
    - startTime: backup job start time
    - finishTime: backup job finish time

PS: finishTime update not work for now, will be fixed in future(metav1.Time in patch)

### ZooKeeperRestore
```yaml
apiVersion: zookeeper.example.com/v1alpha1
kind: ZooKeeperRestore
metadata:
  name: test
  namespace: zookeeper
spec:
  image: go2sheep/zk-helper:latest
  rolloutRestart: true
  source:
    oss:
      endpoint: "oss-cn-shanghai.aliyuncs.com"
      bucket: "zookeeper-test"
      key: "zookeeper/test/once/backup-test-xkntl"
      accessKeySecret:
        name: test-key
        key: accessKey
      secretKeySecret:
        name: test-key
        key: secretKey
  target:
    name: test
    dataDir: /version-2
```
#### Specification
**NOTICE**: the older images have some problems, if you have pulled, please use `image: go2sheep/zk-helper:10cc9d8` to pull new one.
- image: zookeeper restore job image
- rolloutRestart: used to rollout restart statefulset after restore
- source: backup source, must be set
    - oss is alicloud oss, must set endpoint, bucket, key, accessKeySecret and secretKeySecret.
        - key is backup directory in oss
        - can use key in ZooKeeprBackup `status.records`
    - s3 is not supported yet
- target: restore target, must be set
    - can use name or label selector to specify target statefulset
    - dataDir is used to specify zookeeper data directory
#### Status
```yaml
status:
  status: restarted
```
- status: restore status, can be completed, failed, running, pending, restarted
    - restarted: only appear when rolloutRestart is true
- message: restore error message

## Deploy
1. helm charts in git repo
```bash
cd charts/zookeeper-operator

helm install zookeeper-operator ./
```
2. helm charts in charts repo
```bash
helm add repo <alias> https://github.com/ciiiii/helm-charts.git

helm install zookeeper-operator <alias>/zookeeper-operator
```
## Implementation
### zk-manager

entry: cmd/manager/main.go

controllers:

**ClusterController**:
- watch: 
  - ZooKeeperCluster
  - StatefulSet
  - Service
  - ConfigMap
  - Pod(filter with labels)
- reconcile:
  - reoncileGeneric: reconcile resource without special logic
    - Service
    - ConfigMap
    - Role, RoleBinding, ServiceAccount
  - reconcileStatefulset:
    - handling scaling up/down
    - handling image update
  - reconcileStatus
    - parse server status from Pod annotations
  - reconcileFinalizer
    - handling PVC clear

**BackupController**:
- watch:
  - ZooKeeperBackup
  - Job
  - CronJob
- reconcile:
  - reconcileJob/reconcileCronJob
    - choose leader's pvc to backup
    - job is almost immutable
    - cronJob should be updated when spec changes
  - reconcileMessage
    - update error message which shouldn't be requeued
  - reconcileStatus
    - calculate backup status from pods and jobs status

#### RestoreController:
- watch:
  - ZookKeeperRestore
  - Job
- reconcile:
  - reconcileJob
    - restore data to all pod's pvc
    - job is almost immutable
  - reconcileMessage
    - update error message which shouldn't be requeued
  - reconcileStatus
    - calculate restore status from pods and jobs status
    - rollout restart statefulset if necessary
### zk-helper
entry: cmd/helper/main.go

actions:
- init(run in initContainer):
  1. sync myId file
  2. sync dynamic config file: init cluster or join cluster
  3. sync static config
- ready(run by readinessProbe):
  1. check server is ready from remote and local
  2. switch from observer to participant
- watch(run in sidecar):
  1. get server status and patch to Pod annotations
  2. wait for connections close before exit
  3. clear zookeeper server config when server is scaling down
- stop(run by preStopHook):
  1. prevent zookeeper exit directly
- backup(run in backup job):
  1. save zookeeper data to oss
- resotre(run in restore job):
  1. load oss data to zookeeper data directory
## PS
- Test passed on Kubernetes 1.17.12, only support `batchv1beta1.CronJob`, schedule backup will not work in newer version without `batchv1beta1.CronJob`, other features should not be affected.
- There are still some problems about restore which may be related to currentEpoch and acceptedEpoch.