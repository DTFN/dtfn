#!/bin/bash

set -e

function printHelp () {
    echo "Usage: ./`basename $0` -t [up|down]"
    exit 1
}

# parse script args
while getopts ":t:" OPTION; do
    case ${OPTION} in
    t)
        OP_METHOD=$OPTARG
        ;;
    ?)
        printHelp
    esac
done

function validateArgs() {
    if [ -z "${OP_METHOD}" ]; then
        echo "Option up/down not mentioned"
        printHelp
    fi
    
    if [[ "${OP_METHOD}" != "up" 
        && "${OP_METHOD}" != "down" ]]; then
        printHelp
    fi
}


# all global envirment parameter
ROOT_DIR=$(cd `dirname $(readlink -f "$0")`/.. && pwd)
ENV_FILE=${ROOT_DIR}/config/env.json
KEYSTORE=${ROOT_DIR}/config/keystore
GNS_FILE=${ROOT_DIR}/config/genesis.json

DOCKER_OS=$(cat ${ENV_FILE} |jq '.system'|sed 's/"//g')
LOGIN_USR=$(cat ${ENV_FILE} |jq '.user.name'|sed 's/"//g')
LOGIN_PWD=$(cat ${ENV_FILE} |jq '.user.passwd'|sed 's/"//g')
LOCALHOST=$(cat ${ENV_FILE} |jq '.localhost'|sed 's/"//g')
NODE_COUNT=$(cat ${ENV_FILE} |jq '.setup.node.init[]'|wc -l)
NODE_LIST=$(cat ${ENV_FILE} |jq '.setup.node.init[]'|sed 's/"//g')
IP_NUMBER=$(cat ${ENV_FILE} |jq '.setup.node.init[]'|cut -d= -f2|cut -d, -f1|sort|uniq|wc -l)
ADD_COUNT=$(cat ${ENV_FILE} |jq '.setup.node.add.host[]'|wc -l)
ADD_NODES=$(cat ${ENV_FILE} |jq '.setup.node.add.host[]'|sed 's/"//g')
NODE_FROM=$(cat ${ENV_FILE} |jq '.setup.add.from.node'|sed 's/"//g')

PUB_KEYS=${CHAIN_DIR}/pub_keys
NODE_PORT=$(cat ${ENV_FILE} |jq '.setup.port'[])
CHAIN_DIR=$(echo ${NODE_PORT}  |jq '.db'|sed 's/"//g')
INIT_PORT=$(echo ${NODE_PORT} |jq '.ports'|sed 's/"//g')
LOG_LEVEL=$(echo ${NODE_PORT} |jq '.loglevel'|sed 's/"//g')
DBUG_PORT=$(echo ${NODE_PORT} |jq '.debug'|sed 's/"//g')

function sshConn() {
    sshpass -p ${LOGIN_PWD} ssh -o StrictHostKeychecking=no ${LOGIN_USR}@${1} "$2"
}

function copyNodeFiles() {
    echo "copy files: '$2'"
    sshpass -p ${LOGIN_PWD} scp -o StrictHostKeychecking=no -C -r "$2" ${LOGIN_USR}@${1}:${CHAIN_DIR}
}

function createNodeKey() {
    validator=$1
    tagfile=$2

    if [ -f "${tagfile}" ]; then
        rm -f ${tagfile}
    fi

    echo "{\"priv_key\":" >> ${tagfile}
    cat ${validator} |jq -r '.priv_key' >> ${tagfile}
    echo "}" >> ${tagfile}

    cat ${tagfile} |jq . > ${tagfile}.bak
    mv ${tagfile}.bak ${tagfile}
}

function createPubKeyFile() {
    validator=$1
    nodeName=$2

    tmp=${PUB_KEYS}.${nodeName}
    cat ${validator} |jq '.validators[0]' >> ${tmp}

    nodeIdx=$(echo ${nodeName}|awk -Fr '{print $2}')
    if [ ${nodeIdx} -gt 1 ]; then
        sed -i '1i,' ${tmp}
    fi
}

function createStartScript() {
    masterAddr=$1
    masterHost=$2
    name=$3
    ports=$4
    index=$5
    p2Port=$6
    
    startScript=${CHAIN_DIR}/${name}/start.sh
    # echo "test node: ${IP_NUMBER}"
    # docker network mode, single use bridge
    if [ "${IP_NUMBER}" -gt 1 ]; then network_mode=host; else network_mode=bridge; fi

    # persistent peers, and support pprof debug port
    pprof_debug=""
    persistent_peers=""
    if [ "${index}" -gt 0 ]; then persistent_peers="--persistent_peers=${masterAddr}@${masterHost}:46656"; fi
    if [ "${masterHost}" == "${LOCALHOST}" ]; then
        if [ "${DBUG_PORT}" -ne 0 ]; then
            DBUG_PORT=$(expr ${DBUG_PORT} + 1);
            ports="${argPorts} -p ${DBUG_PORT}:${DBUG_PORT}"
            pprof_debug="--pprof_port=${DBUG_PORT}";
        fi
    fi

cat << EOF > ${startScript}
docker run -tid --net=${network_mode} --name=${name} \\
    ${argPorts} \\
    -v ${CHAIN_DIR}/${name}:/chaindata \\
    -v /usr/bin:/bin ${DOCKER_OS} /bin/dtfn \\
    --datadir /chaindata --rpcapi eth,net,web3,personal,admin,shh --with-tendermint --rpc \\
    --rpccorsdomain=* --rpcvhosts=* --rpcaddr=0.0.0.0 --ws --wsaddr=0.0.0.0 --gcmode=full --lightpeers=15 \\
    --pex=true --fast_sync=true --routable_strict=false --trie_time_limit=1 \\
    --need_proof_block=false --tm_cons_emptyblock=true --tm_cons_eb_inteval=10 \\
    --priv_validator_file=config/priv_validator.json --addr_book_file=addr_book.json --initial_eth_account=config/initial_eth_account.json \\
    --tendermint_p2paddr=tcp://0.0.0.0:${p2Port} ${persistent_peers} ${pprof_debug} --logLevel=${LOG_LEVEL}
EOF

    # init user keystore
    peer_keystore=${CHAIN_DIR}/${name}/keystore
    if [[ -d "${KEYSTORE}" && -d "${peer_keystore}" ]]; then
        sudo rm -rf ${peer_keystore}/*
        sudo cp -r ${KEYSTORE}/* ${peer_keystore}/
    fi
}

function mergeNodePubKeys() {
    echo "start merge node pubkeys ..."
    temp=${PUB_KEYS}.tmp
    echo "{\"validators\":[" >> ${temp}
    cat $(ls ${PUB_KEYS}.peer*) >> ${temp}
    echo "]}" >> ${temp}
    cat ${temp} |jq . > ${PUB_KEYS}

    rm -rf ${temp} ${PUB_KEYS}.peer*
}

function replacePubKey() {
    validator=$1
    replaceStr="$2,"
    chainid=$3

    cat ${validator} |jq . > ${validator}.bak
    value=$(sed -n '/app_hash/=' ${validator}.bak)
    start=$(sed -n '/validators/=' ${validator}.bak)
    end=$(expr "${value}" - 1)
    sed "${start},${end}c $(echo ${replaceStr})" ${validator}.bak |jq . > ${validator}

    oldValue=$(cat ${validator} |jq -r '.chain_id')
    sed -i "s/${oldValue}/${chainid}/g" ${validator}

    rm -f ${validator}.bak
}

function replaceGenesisPubKey() {
    echo "start replace genesis pubkeys ..."
    chainid=$1
    end_mark=$(expr $(sed -n '$=' ${PUB_KEYS}) - 1)
    context=$(sed -n "2,${end_mark}p" ${PUB_KEYS})

    for node in ${NODE_LIST}; do
        nodeInfo=${node%,*}
        name=$(echo ${nodeInfo%=*})

        json_file=${CHAIN_DIR}/${name}/tendermint/config/genesis.json
        replacePubKey ${json_file} "${context}" "${chainid}"
    done
    rm -f ${PUB_KEYS}
}

function startNodeService() {
    for node in ${NODE_LIST}; do
        nodeInfo=${node%,*}
        name=${nodeInfo%=*}
        addr=${nodeInfo#*=}

        echo "start script at ${addr}:${name} ..."
        if [ "${addr}" == "${LOCALHOST}" ]; then
            sh ${CHAIN_DIR}/${name}/start.sh
        else
            sshConn ${addr} "mkdir -p ${CHAIN_DIR}"
            copyNodeFiles ${addr} "${CHAIN_DIR}/${name}"
            copyNodeFiles ${addr} "${ROOT_DIR}/config"
            copyNodeFiles ${addr} "${ROOT_DIR}/script"
            copyNodeFiles ${addr} "${ROOT_DIR}/tools"
            sshConn ${addr} "sh ${CHAIN_DIR}/${name}/start.sh"
        fi
    done
}

function adjustLocalPortOfStartCommand() {
    addValue=0
    if [ "${1}" == "${LOCALHOST}" ]; then
        addValue=${2}
    fi

    adjusted=""
    for p in ${INIT_PORT}; do
        value=$(echo ${p} |awk -F':' -v vl="${addValue}" '{print "-p "$1+vl":"$2}')
        adjusted="${adjusted} ${value}"
    done
    
    echo ${adjusted}
}

function networkUp() {
    testnet=${ROOT_DIR}/script/mytestnet
    sudo rm -rf ${CHAIN_DIR}/* ${testnet}
    dtfn testnet ${NODE_COUNT}
    chmod -R 0755 ${testnet}
    
    master_address=""
    master_hostip=""
    master_chainid=""
   
    index=0
    for node in ${NODE_LIST}; do
        nodeInfo=${node%,*}
        typeInfo=${node#*,}
        
        name=${nodeInfo%=*}
        addr=${nodeInfo#*=}
        nodeType=${typeInfo#*=}
        argPorts=$(adjustLocalPortOfStartCommand ${addr} $(expr ${name#*r} - 0))
        home=${CHAIN_DIR}/${name}/tendermint
        chaindata=${CHAIN_DIR}/${name}/dtfn/chaindata

        dtfn --datadir ${CHAIN_DIR}/${name}/ init ${GNS_FILE}
        cp ${GNS_FILE} ${chaindata}
        mkdir -p ${home} && cp -r ${testnet}/node${index}/config ${home}/
        ethAccount ${GNS_FILE}
        mv ${ROOT_DIR}/script/initial_eth_account.json ${home}/config/
        sed -i 's/node/peer/g' ${home}/config/genesis.json

        createNodeKey ${home}/config/priv_validator.json ${home}/config/node_key.json
        if [ "${index}" -eq 0 ]; then
            master_host=$addr
            master_address=$(cat ${home}/config/priv_validator.json |jq -r '.address' |sed 's/"//g' |tr 'A-Z' 'a-z')
            master_chainid=$(cat ${home}/config/genesis.json |jq -r '.chain_id')
        fi
        p2pAddrPort=$(echo $argPorts|awk -F'-p' '{print $3}'|cut -d':' -f1)
        # if [ "${master_host}" == "${LOCALHOST}" ]; then p2pAddrPort=$(expr $p2pAddrPort + 1) ; fi
        createStartScript ${master_address} ${master_host} ${name} "${argPorts}" $index $p2pAddrPort

        index=$(expr $index + 1)
    done
    sudo rm -rf ${testnet}

    startNodeService
}

function networkDown() {
    for node in ${NODE_LIST}; do
        nodeInfo=${node%,*}
        addr=${nodeInfo#*=}

        echo "stop network at ${addr} ..."
        if [ "${addr}" != "${LOCALHOST}" ]; then
            sshConn ${addr} "docker ps -a |grep dtfn |awk '{print \$1}' |xargs -ti docker stop {}"
            sshConn ${addr} "docker ps -a |grep dtfn |awk '{print \$1}' |xargs -ti docker rm -f {}"
            sshConn ${addr} "rm -rf ${CHAIN_DIR}/*"
        else
            docker ps -a |grep dtfn |awk '{print $1}' |xargs -ti docker stop {} >/dev/null 2>&1
            docker ps -a |grep dtfn |awk '{print $1}' |xargs -ti docker rm -f {}
            break
        fi
    done
}

function networkAdd() {
    first_node=$(cat ${ENV_FILE} |grep '^init:peer' |cut -d: -f2 |head -1)
    first_node_name=${first_node%=*}
    first_node_host=${first_node#*=}
    home=${CHAIN_DIR}/${first_node_name}/tendermint
    if [ ! -d "${home}" ]; then
        echo "${first_node_name} is not exists"
        exit 1
    fi
    master_address=$(cat ${home}/config/priv_validator.json |jq -r '.address' |sed 's/"//g' |tr 'A-Z' 'a-z')

    testnet=${ROOT_DIR}/script/mytestnet
    dtfn testnet ${ADD_COUNT}
    chmod -R 0755 ${testnet}

    index=0
    for node in ${ADD_NODES}; do
        nodeInfo=${node%,*}
        typeInfo=${node#*,}
        
        name=${nodeInfo%=*}
        addr=${nodeInfo#*=}
        nodeType=${typeInfo#*=}
        argPorts=$(adjustLocalPortOfStartCommand ${addr} $(expr ${name#*r} - 0))
        home=${CHAIN_DIR}/${name}/tendermint
        chaindata=${CHAIN_DIR}/${name}/dtfn/chaindata

        dtfn --datadir ${CHAIN_DIR}/${name}/ init ${GNS_FILE}
        cp ${GNS_FILE} ${chaindata}
        mkdir -p ${home} && cp -r ${testnet}/node${index}/config ${home}/
        ethAccount ${GNS_FILE}
        mv ${ROOT_DIR}/script/initial_eth_account.json ${home}/config/
        sed -i 's/node/peer/g' ${home}/config/genesis.json
        
        createNodeKey ${home}/config/priv_validator.json ${home}/config/node_key.json
        p2pAddrPort=$(echo $argPorts|awk -F'-p' '{print $3}'|cut -d':' -f1)
        # if [ "${first_node_host}" == "${LOCALHOST}" ]; then p2pAddrPort=$(expr $p2pAddrPort + 1) ; fi
        createStartScript ${master_address} ${first_node_host} ${name} "${argPorts}" 1 $p2pAddrPort
       
        if [ "${NODE_FROM}" != "" ]; then
            echo "only support repair one consensus node"
            break
        fi
        index=$(expr $index + 1)
    done
    sudo rm -rf ${testnet}
}

function main() {
    case ${OP_METHOD} in
        "up")
            networkUp
            ;;
        "down")
            networkDown
            ;;
        "add")
            networkAdd
            ;;
    esac
}
validateArgs
main
