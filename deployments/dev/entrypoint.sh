#!/usr/bin/env sh

export REPLICA_ID=$(nslookup $(hostname -i) | grep in-addr.arpa | awk '{ print $4 }' | tr -d -c 0-9)

exec /bin/keepalived-exporter \
    --container-name ${COMPOSE_PROJECT_NAME}-keepalived-${REPLICA_ID} \
    --container-tmp-dir /tmp/keepalived-tmp/ \
    --cs /keepalived-exporter-cs.sh
