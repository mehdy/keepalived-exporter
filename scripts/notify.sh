#!/bin/bash

set_variables() {
    readonly internet_device=$1
    readonly backup_address=$2
    readonly mode=$3
}

setup_device_ip() {
    if [ "${mode}" == "MASTER" ]
    then
        ip address del ${backup_address}/24 dev ${internet_device}
    else
        ip address add ${backup_address}/24 dev ${internet_device}
    fi
}

main() {
    set_variables $@
    setup_device_ip
}

main $1 $2 $5
