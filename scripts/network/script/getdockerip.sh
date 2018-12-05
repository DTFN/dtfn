#!/bin/bash

set -e

# all global envirment parameter
ROOT_DIR=$(cd `dirname $(readlink -f "$0")`/.. && pwd)
ENV_FILE=${ROOT_DIR}/config/env.json
CHAIN_DIR=$(cat ${ENV_FILE} |jq '.setup.port'[]|jq '.db'|sed 's/"//g')

function get_docker_ip() {
    ip=$(docker inspect --format '{{ .NetworkSettings.IPAddress }}' $1)
    echo $ip
}

function get_node_address() {
    nodeFile=${CHAIN_DIR}/$1/tendermint/config/priv_validator.json
    address=$(sudo jq '.address' $nodeFile |sed 's/"//g' |tr 'A-Z' 'a-z')
    echo $address
}

## function main
function main() {
    contains=$(docker ps -a --format '{{.Names}}' |sort)
    for i in ${contains}; do
        ipAddress=$(get_docker_ip $i)
        nodeAddress=$(get_node_address $i)
        echo "$i $ipAddress $nodeAddress"
    done
}
main
