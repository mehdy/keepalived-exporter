#!/bin/bash

main() {
    log_message ${LOG_LEVEL_DEBUG} "adding user '${SERVICE_USER_NAME}' to all groups"

    for group_name in ${SERVICE_GROUP_NAMES}
    do
        usermod -a -G ${group_name} ${SERVICE_USER_NAME}
        ret_val=$?
        if [ ${ret_val} -ne 0 ]
        then
            return ${ret_val}
        fi
    done

    log_message ${LOG_LEVEL_DEBUG} "done adding user '${SERVICE_USER_NAME}' to all groups"
    return 0
}

main
