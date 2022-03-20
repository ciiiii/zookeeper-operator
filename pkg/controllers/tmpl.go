package controllers

const (
	initCommand = `set -ex
/usr/local/bin/zk-helper --action=init
cp /usr/local/bin/zk-helper %[1]s
chmod a+x %[1]s
cp -f %[2]s/%[4]s %[3]s
cp -f %[2]s/%[5]s %[3]s`

	zooCfg = `4lw.commands.whitelist=cons, envi, conf, crst, srvr, stat, mntr, ruok
dataDir=%[1]s
standaloneEnabled=false
reconfigEnabled=true
skipACL=yes
initLimit=10
syncLimit=2
tickTime=2000
globalOutstandingLimit=1000
preAllocSize=65536
snapCount=10000
commitLogCount=500
snapSizeLimitInKb=4194304
maxCnxns=0
maxClientCnxns=60
minSessionTimeout=4000
maxSessionTimeout=40000
autopurge.snapRetainCount=3
autopurge.purgeInterval=1
quorumListenOnAllIPs=false
admin.serverPort=8080
dynamicConfigFile=%[1]s/%[2]s`

	log4jCfg = `zookeeper.root.logger=CONSOLE
zookeeper.console.threshold=INFO
log4j.rootLogger=${zookeeper.root.logger}
log4j.appender.CONSOLE=org.apache.log4j.ConsoleAppender
log4j.appender.CONSOLE.Threshold=${zookeeper.console.threshold}
log4j.appender.CONSOLE.layout=org.apache.log4j.PatternLayout
log4j.appender.CONSOLE.layout.ConversionPattern=%d{ISO8601} [myid:%X{myid}] - %-5p [%t:%C{1}@%L] - %m%n`

	log4jQuietCfg = `log4j.rootLogger=ERROR, CONSOLE
log4j.appender.CONSOLE=org.apache.log4j.ConsoleAppender
log4j.appender.CONSOLE.Threshold=ERROR
log4j.appender.CONSOLE.layout=org.apache.log4j.PatternLayout
log4j.appender.CONSOLE.layout.ConversionPattern=%d{ISO8601} [myid:%X{myid}] - %-5p [%t:%C{1}@%L] - %m%n
`
)
