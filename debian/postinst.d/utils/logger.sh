#!/bin/bash

# Function to log a message to the log file
function log_message {
    # Early exit if QUIET mode is on
    if $QUIET
    then
        return
    fi

    local level="$1"
    local message="$2"
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    if [ "$level" -ge "${LOG_LEVEL_MIN}" ]
    then
        echo "[${timestamp}] [${LOG_LEVELS_ARR[level]}] ${message}"
    fi
}
