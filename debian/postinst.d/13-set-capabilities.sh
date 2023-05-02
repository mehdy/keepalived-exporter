#!/bin/bash

main() {
    log_message ${LOG_LEVEL_DEBUG} "set capabilities to binary"

    # Give capabilities to signal keepalived, and read the output files
    setcap "CAP_DAC_READ_SEARCH,CAP_KILL=+eip" /opt/ottopia/keepalived-exporter/keepalived-exporter
    local RET_VAL=$?
    if [ ${RET_VAL} -ne 0 ]
    then
        log_message ${LOG_LEVEL_ERROR} "Error when setting cap. look at above output"
        exit ${RET_VAL}
    fi

    log_message ${LOG_LEVEL_DEBUG} "Done set capabilities to binary"
}

main
