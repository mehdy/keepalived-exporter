#!/usr/bin/env sh

echo "starting keepalived"
keepalived --log-console --log-detail

echo "starting endpoint"
shell2http -form /signal 'kill -s $v_signal $(cat /var/run/keepalived/keepalived.pid)' /signal/num 'keepalived --signum $v_signal' /version 'keepalived -v 2>&1'
