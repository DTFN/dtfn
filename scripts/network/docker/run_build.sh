#!/bin/bash

set -e

# all global envirment parameter
OS_ARCH=$(uname -n|tr '[:upper:]' '[:lower:]')
ROOT_DIR=$(cd `dirname $(readlink -f "$0")` && pwd)

function printHelp () {
    echo "Usage: ./`basename $0` -t [build|clean]"
    exit 1
}

# parse script args
while getopts ":t:" OPTION; do
    case ${OPTION} in
        t)
            OP_COMMAND=$OPTARG
            ;;
        ?)
            printHelp
    esac
done

function validateArgs() {
    if [[ "${OP_COMMAND}" != "build" 
        && "${OP_COMMAND}" != "clean" ]]; then
        printHelp
    fi
}

function do_build() {
    current=$(pwd)
    imageName="gelchain:18.04" 
    
    if [[ "${OS_ARCH}" == "centos" ]]; then
        docker build -f dockerfile/centos -t "centos/${imageName}" .
    elif [[ "${OS_ARCH}" == "mint" || "${OS_ARCH}" == "ubuntu" ]]; then
        docker build -f dockerfile/ubuntu -t "ubuntu/${imageName}" .
    # elif [[ "${OS_ARCH}" =~ "macbook" ]]; then
        # docker build -f dockerfile/draw -t "draw/${imageName}" .
    fi
}

function do_clean() {
    docker images |grep gelchain |awk '{print $3}' |xargs -ti docker rmi {}
}

function main() {
    case ${OP_COMMAND} in
        "build")
            do_build
            ;;
        "clean")
            do_clean
            ;;
    esac
}
validateArgs
main
