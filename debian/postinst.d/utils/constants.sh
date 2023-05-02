#!/bin/bash

# General configurations
readonly SERVICE_NAME="ottopia-keepalived-exporter"
readonly SERVICE_USER_NAME="ottopia_keepalived"
readonly SERVICE_GROUP_NAMES="ottopia_keepalived"

# logger.sh configs
QUIET=false

# Define log levels
readonly LOG_LEVEL_DEBUG=0
readonly LOG_LEVEL_INFO=1
readonly LOG_LEVEL_WARN=2
readonly LOG_LEVEL_ERROR=3

OIFS=${IFS}
IFS=","
readonly LOG_LEVELS="DEBUG,INFO,WARN,ERROR"
readonly LOG_LEVELS_ARR=(${LOG_LEVELS})
IFS=${OIFS}
# Set the default log level
if [ "${LOG_LEVEL_MIN}" = "" ]
then
    LOG_LEVEL_MIN=$LOG_LEVEL_INFO
fi
