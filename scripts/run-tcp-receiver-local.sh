#!/bin/bash

log()
{
	logTime=`gdate +"%s.%N"`
	echo "${logTime},SERVERLOG,$1"
}

verifyReconfiuration() 
{
	while true; do
		nc -z $1 $2
		if [ $? -eq 0 ]; then
			log "Done reconfiguration of proxy with frontend port: $2"
			break
		fi
	done
	processId=$3	
	processCount=`ps -ef | grep "${processId}" | grep -v grep | wc -l`
	log "Number of proxy processes: $processCount"	
}

if [ $# -ne 5 ]; then	
	echo "Usage: $0 <number of app instances> <start frontend port> <start backend port> <sleep time> <haproxy|nginx>"
	echo "Example: $0 3 1234 3333 1 haproxy"
	echo "Example: $0 3 1234 3333 1 nginx"
	exit 1
fi

NUM_INSTANCES=$1
START_FRONTEND_PORT=$2
START_BACKEND_PORT=$3
SLEEP_TIME=$4
PROXY=$5
if [ $PROXY != "haproxy" -a  $PROXY != "nginx" ]; then
	echo "Only haproxy and nginx are supported as possible options"
	exit 2
fi
HAPROXY_FORMAT="listen %s\n  mode tcp\n  bind :%s\n  server %s\n"
NGINX_FORMAT="stream {\n  upstream %s {\n    server %s;\n  }\n  server {\n    listen %s;\n    proxy_pass %s;\n  }\n}\n"
DIR=`dirname $0`
pushd $DIR
HAPROXY_CONFIG=$PWD/haproxy.cfg
NGINX_CONFIG=$PWD/nginx.conf
popd
PID_FILE=$DIR/haproxy.pid
if [ $PROXY = "haproxy" ]; then
	cp $DIR/haproxy.cfg.template $HAPROXY_CONFIG
	log "Killing existing haproxy process"
	killall haproxy
	log "Starting haproxy"
	haproxy -f "${HAPROXY_CONFIG}" -D -p "${PID_FILE}"
elif [ $PROXY = "nginx" ]; then
	cp $DIR/nginx.conf.template $NGINX_CONFIG
	log "Killing existing nginx process"
	nginx -c "${NGINX_CONFIG}" -s quit
	log "Starting nginx"
	nginx -c "${NGINX_CONFIG}"
fi
sleep $SLEEP_TIME
count=0
log "Killing existing tcp-sample-receiver processes"
killall tcp-sample-receiver
for i in `seq 1 $NUM_INSTANCES`
do
	backendPort=`expr $START_BACKEND_PORT + $count`
	frontendPort=`expr $START_FRONTEND_PORT + $count`
	count=`expr $count + 1`
	log "Starting process on 127.0.0.1:$backendPort"
	tcp-sample-receiver -address 127.0.0.1:$backendPort &		
	log "Starting reconfiguration of proxy with frontend port: $frontendPort"
	if [ $PROXY = "haproxy" ]; then
		printf "${HAPROXY_FORMAT}" "proxy$i" "${frontendPort}" "app$i 127.0.0.1:$backendPort" >> "${HAPROXY_CONFIG}" 	
		haproxy -f "${HAPROXY_CONFIG}" -p "${PID_FILE}" -D -sf $(cat $PID_FILE)
		verifyReconfiuration "127.0.0.1" $frontendPort $HAPROXY_CONFIG
	elif [ $PROXY = "nginx" ]; then
		printf "${NGINX_FORMAT}" "stream_backend_$i" "127.0.0.1:$backendPort" "${frontendPort}" "stream_backend_$i" >> "${NGINX_CONFIG}" 	
		nginx -c $NGINX_CONFIG -s reload
		verifyReconfiuration "127.0.0.1" $frontendPort "nginx: worker"
	fi	
	log "Sleeping for ${SLEEP_TIME} seconds"
	sleep $SLEEP_TIME
done
