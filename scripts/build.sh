#!/bin/bash

set -e

ROOT_DIR=$(cd `dirname $(readlink -f "$0")`/.. && pwd)
OS_ARCH=$(uname -n|tr '[:upper:]' '[:lower:]')
BUILD_FLAGS="-ldflags \"-X github.com/green-element-chain/gelchain/version.GitCommit=\`git rev-parse --short HEAD\`\""
BUILD_TAGS=gelchain

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
    elif [[ "${OS_ARCH}" == "mint" || "${OS_ARCH}" == "ubuntu" ]]; then
        sudo apt-get install -y libgmp-dev libssl-dev openssl gcc
    elif [[ "${OS_ARCH}" =~ "macbook" ]]; then
        sudo brew install gmp openssl gcc
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

function do_executeCmd() {
    echo "do execute command for gelchain"
    # do_getDependencies
    do_blsPackageCmd 
    
    echo $1
    cd ${ROOT_DIR} && sh -c "$1"

    tagbin=${ROOT_DIR}/${BUILD_TAGS}
    if [[ -f "${tagbin}" ]]; then
        sudo mv ${tagbin} /usr/bin/${BUILD_TAGS}
    fi
}

function do_remove_file() {
    if [ -f "$1" ]; then
        sudo rm -f $1
    fi
}

function do_clean() {
    echo "do clean gelchain"
    do_blsPackageCmd 'clean'
   
    cd ${ROOT_DIR}
    do_remove_file ${BUILD_TAGS}
    do_remove_file /usr/bin/${BUILD_TAGS}
    do_remove_file ${GOPATH}/bin/${BUILD_TAGS}
}

function main() {
    current=$(pwd)
    case ${OP_TAGET} in
        "blsdep")
            do_blsDependencies 2>&1 >/dev/null
            ;;
        "glide")
            do_getDependencies
            ;;
        "build")
            shellCmd="CGO_ENABLED=1 go build ${BUILD_FLAGS} -o ./${BUILD_TAGS} ./cmd/gelchain"
            do_executeCmd "${shellCmd}"
            ;;
        "install")
            shellCmd="CGO_ENABLED=1 go install ${BUILD_FLAGS} ./cmd/gelchain"
            do_executeCmd "${shellCmd}"
            ;;
        "clean")
            do_clean
            ;;
    esac
    cd ${current}
}
validateArgs
main
