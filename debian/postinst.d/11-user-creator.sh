#!/bin/bash

main() {
    getent passwd "${SERVICE_USER_NAME}" &>/dev/null
    if [ $? -eq 0 ]
    then
        log_message ${LOG_LEVEL_INFO} "User '${SERVICE_USER_NAME}' exists"
        return 0
    fi

    OIFS=${IFS}
    IFS=','

    local group_name=(${SERVICE_GROUP_NAMES})
    adduser --system --no-create-home --ingroup ${group_name} ${SERVICE_USER_NAME}

    IFS=${OIFS}
    return $?
}

main
