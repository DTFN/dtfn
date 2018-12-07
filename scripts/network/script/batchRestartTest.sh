#!/bin/bash

set -e

# all global envirment parameter
ROOT_DIR=$(cd `dirname $(readlink -f "$0")`/.. && pwd)
SHEL_DIR=${ROOT_DIR}/script
ENV_FILE=${ROOT_DIR}/config/env.json
NODE_PORT=$(cat ${ENV_FILE} |jq '.setup.port'[])
CHAIN_DIR=$(echo ${NODE_PORT}  |jq '.db'|sed 's/"//g')

LAST_HEIGHT=1
MAX_ROUND_TEST=0

function printHelp () {
    echo "Usage: ./`basename $0` -n [round]"
    exit 1
}

# parse script args
while getopts ":n:" OPTION; do
    case ${OPTION} in
    n)
        MAX_ROUND_TEST=$OPTARG
        ;;
    ?)
        printHelp
    esac
done

function validateArgs() {
    if [[ "${MAX_ROUND_TEST}" == "" 
        || "${MAX_ROUND_TEST}" -le 0 ]]; then
        MAX_ROUND_TEST=10
    fi
}

function message_color() {
    echo -e "\033[40;31m[$1]\033[0m"
}

function kill_monitor() {
    docker ps -aq |xargs -ti docker rm -f {} > /dev/null 2>&1
    ps -ef |grep 'docker logs' |grep -v grep |awk '{print $2}' |xargs -ti kill -9 {}
}

function get_current_block_height() {
    logline=$(docker logs peer1 |grep 'Finalizing commit of block'|tail -1)
    if [ "${logline}" == "" ]; then
        echo "last round not new block"
        exit 1
    fi
    echo ${logline}
    LAST_HEIGHT=$(echo "'"${logline}"'" |awk -F'=' '{print $3}'|awk '{print $1}')
    message_color "after run 30 second, the current height is ${LAST_HEIGHT}"
    kill_monitor
}

function random() {
    min=$1
    max=$2-$1
    num=$(head -200 /dev/urandom |cksum|cut -d' ' -f1)
    ((retnum=num%max+min))
    echo $retnum
}

function monitor_block() {
    sleepTime=$(random 30 120)
    echo "monitor block sleep time: ${sleepTime}"
    sleep "${sleepTime}"

    docker ps -aq |xargs -ti docker stop {} > /dev/null 2>&1
}

function create_new_chain() {
    kill_monitor
    
    current=$(pwd)
    cd ${ROOT_DIR} && make up
    cd ${current}

    monitor_block
}

function check_socket_port() {
    while [ "$(netstat -na |grep '46656')" != "" ]; do
        sleep 1
    done
}

function restart_chain() {
    check_socket_port

    peers=$(find ${CHAIN_DIR} -maxdepth 1 -name "peer*")
    for p in ${peers}; do
        sh ${p}/start.sh
    done
}

function round_test() {
    get_current_block_height

    # do nothing
    restart_chain
    monitor_block
}

# main function
function main() {
    message_color "You want test round ${MAX_ROUND_TEST}"
    if [[ "${MAX_ROUND_TEST}" -le 0 ]]; then
        message_color "test round must > 0"
        exit 1
    fi
    
    create_new_chain
    declare -i i=1
    while ((i<=${MAX_ROUND_TEST})); do
        message_color "execute test round ${i}"
        round_test
        let ++i
    done
    get_current_block_height
}
validateArgs
main
