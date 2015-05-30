#!/bin/bash

log()
{
	logTime=`gdate +"%s.%N"`
	echo "${logTime},CLIENTLOG,$1"
}

verifyProxyProcesses() 
{
	processId=$1
	processGrepId=$2
	desiredCount=$3
	log "Monitoring $processId count"
	while true; do
		processCount=`ps -ef | grep "${processGrepId}" | grep -v grep | wc -l`		
		if [ $processCount -eq $desiredCount ]; then
			log "Number of proxy $processId processes: $processCount"
			break
		fi
	done		
}

if [ $# -ne 6 ]; then	
	echo "Usage: $0 <server address> <start frontend port> <number of frontend ports> <number of concurrent connections per port> <number of virtual users> <haproxy|nginx>"
	echo "Example: $0 127.0.0.1 1234 3 3 5 haproxy"
	echo "Example: $0 127.0.0.1 1234 3 3 5 nginx"
	exit 1
fi

SERVER_ADDRESS=$1
START_FRONTEND_PORT=$2
PORT_SPAN=$3
NUM_CONCURRENT_CONNECTIONS=$4
NUM_VIRTUAL_USERS=$5
DIR=`dirname $0`
PROXY=$6
if [ $PROXY != "haproxy" -a  $PROXY != "nginx" ]; then
	echo "Only haproxy and nginx are supported as possible options"
	exit 2
fi
tcp-sample-sender -address $SERVER_ADDRESS -startPort $START_FRONTEND_PORT -portSpan $PORT_SPAN -concurrentConnections $NUM_CONCURRENT_CONNECTIONS -virtualUsers $NUM_VIRTUAL_USERS
log "Finished tcp-sample-sender"

if [ $PROXY = "haproxy" ]; then
	HAPROXY_CONFIG=$DIR/haproxy.cfg
	verifyProxyProcesses $PROXY $HAPROXY_CONFIG 1
elif [ $PROXY = "nginx" ]; then
	NGINX_CONFIG=$DIR/nginx.conf
	verifyProxyProcesses $PROXY "nginx: worker" 3
fi