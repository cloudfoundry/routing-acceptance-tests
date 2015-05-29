#!/bin/bash
if [ $# -ne 4 ]; then
	echo "Usage: $0 <number of app instances> <start frontend port> <start backend port> <sleep time>"
	echo "Example: $0 3 1234 3333 1"
	exit 1
fi

NUM_INSTANCES=$1
START_FRONTEND_PORT=$2
START_BACKEND_PORT=$3
SLEEP_TIME=$4
FORMAT="listen %s\n  mode tcp\n  bind :%s\n  server %s\n"
DIR=`dirname $0`
CONFIG=$DIR/haproxy.cfg
PID_FILE=$DIR/haproxy.pid
cp $DIR/haproxy.cfg.template $CONFIG
echo "Killing existing haproxy process"
killall haproxy
echo "Starting haproxy"
haproxy -f "${CONFIG}" -D -p "${PID_FILE}"
sleep $SLEEP_TIME
count=0
echo "Killing existing tcp-sample-receiver processes"
killall tcp-sample-receiver
for i in `seq 1 $NUM_INSTANCES`
do
	backendPort=`expr $START_BACKEND_PORT + $count`
	frontendPort=`expr $START_FRONTEND_PORT + $count`
	count=`expr $count + 1`
	echo "Starting process on 127.0.0.1:$backendPort"
	tcp-sample-receiver -address 127.0.0.1:$backendPort &
	printf "${FORMAT}" "proxy$i" "${frontendPort}" "app$i 127.0.0.1:$backendPort" >> "${CONFIG}" 
	haproxy -f "${CONFIG}" -p "${PID_FILE}" -D -sf $(cat $PID_FILE)
	echo "Sleeping for ${SLEEP_TIME} seconds"
	sleep $SLEEP_TIME
done