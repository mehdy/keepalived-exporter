#!/bin/bash

create_services_group() {
    local group_name=$1
    getent group ${group_name} >& /dev/null
    if [ $? -eq 0 ]; then
        log_message ${LOG_LEVEL_DEBUG} "Services group '${group_name}' exists."
        return 0
    fi

    addgroup --system "${group_name}"
    return $?
}

main() {
    log_message ${LOG_LEVEL_DEBUG} "creating all groups"

    OIFS=${IFS}
    IFS=','

    for group_name in ${SERVICE_GROUP_NAMES}; do
        create_services_group ${group_name}
        local ret_val=$?
        if [ ${ret_val} -ne 0 ]; then
            log_message ${LOG_LEVEL_ERROR} "Failed to create group '${group_name}' for the services."
            return ${ret_val}
        fi
    done

    IFS=${OIFS}

    log_message ${LOG_LEVEL_DEBUG} "done creating all groups"
}

main
