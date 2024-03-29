#!/bin/bash

set -e

ROOT_DIR=$(cd `dirname $(readlink -f "$0")`/.. && pwd)
OS_ARCH=$(cat /etc/os-release |grep '^ID='|cut -d'=' -f2|sed 's/"//g'|tr '[:upper:]' '[:lower:]')
BUILD_FLAGS="-ldflags \"-X github.com/DTFN/dtfn/version.GitCommit=\`git rev-parse --short HEAD\`\""
BUILD_TAGS=dtfn
ETH_ACCOUNT=ethAccount

function printHelp () {
    echo "Usage: ./`basename $0` -t [blsdep|glide|build|install|clean]"
    exit 1
}

# parse script args
while getopts ":t:" OPTION; do
    case ${OPTION} in
        t)
            OP_TAGET=$OPTARG
            ;;
        ?)
            printHelp
    esac
done

function validateArgs() {
    if [[ "${OP_TAGET}" != "blsdep" 
        && "${OP_TAGET}" != "glide" 
        && "${OP_TAGET}" != "build" 
        && "${OP_TAGET}" != "install" 
        && "${OP_TAGET}" != "clean" ]]; then
        printHelp
    fi
}

function do_getDependencies() {
    echo "do glide install ..."
    if [ "$(which glide)" == "" ]; then
        curl https://glide.sh/get |sh
        glide install
    fi
    glide up
}

function do_blsDependencies() {
    echo "do install bls dependencies ..."
    if [[ "${OS_ARCH}" == "centos" ]]; then
        sudo yum install -y gmp-devel openssl-devel gcc
    elif [[ "${OS_ARCH}" == "linuxmint" || "${OS_ARCH}" == "ubuntu" ]]; then
        sudo apt-get install -y libgmp-dev libssl-dev openssl gcc
    else
        echo "not support, maybe it is a macbook"
        # sudo brew install -y gmp openssl gcc
    fi
}

function do_blsPackageCmd() {
    pkg_herumi=${ROOT_DIR}/vendor/github.com/herumi
    exeCmd="make"
    if [ ! -z "$1" ]; then
        exeCmd="${exeCmd} clean"
    fi
    cd ${pkg_herumi}/mcl && ${exeCmd}
    cd ${pkg_herumi}/bls && ${exeCmd}
}

function do_move_file() {
    if [ -f "$1/$2" ]; then
        sudo mv $1/$2 /usr/bin/$2
    fi
}

function do_executeCmddtfn() {
    echo "do execute command for dtfn ..."
    do_blsPackageCmd 
    
    echo $1
    cd ${ROOT_DIR} && sh -c "$1"
    do_move_file ${ROOT_DIR} ${BUILD_TAGS}
}

function do_executeCmdAccount() {
    echo "do execute command for account ..."
    
    echo $1
    cd ${ROOT_DIR} && sh -c "$1"
    do_move_file ${ROOT_DIR} ${ETH_ACCOUNT}
}

function do_delete_file() {
    if [ -f "$1" ]; then
        sudo rm -f $1
    fi
}

function do_clean() {
    echo "do clean dtfn"
    do_blsPackageCmd 'clean'
   
    cd ${ROOT_DIR}
    do_delete_file ${BUILD_TAGS}
    do_delete_file /usr/bin/${BUILD_TAGS}
    do_delete_file ${GOPATH}/bin/${BUILD_TAGS}
    
    do_delete_file ${ETH_ACCOUNT}
    do_delete_file /usr/bin/${ETH_ACCOUNT}
    do_delete_file ${GOPATH}/bin/${ETH_ACCOUNT}
}

function main() {
    current=$(pwd)
    case ${OP_TAGET} in
        "blsdep")
            do_blsDependencies
            ;;
        "glide")
            do_getDependencies
            ;;
        "build")
            do_executeCmddtfn "CGO_ENABLED=1 go build ${BUILD_FLAGS} -o ./${BUILD_TAGS} ./cmd/dtfn"
            do_executeCmdAccount "CGO_ENABLED=1 go build ${BUILD_FLAGS} -o ./${ETH_ACCOUNT} ./cmd/ethAccount"
            ;;
        "install")
            do_executeCmddtfn "CGO_ENABLED=1 go install ${BUILD_FLAGS} ./cmd/dtfn}"
            do_executeCmdAccount "CGO_ENABLED=1 go install ${BUILD_FLAGS} ./cmd/ethAccount}"
            ;;
        "clean")
            do_clean
            ;;
    esac
    cd ${current}
}
validateArgs
main
