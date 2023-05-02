#!/bin/bash

main() {
    env python3.9 -m pip install -r /opt/ottopia/keepalived-exporter/scripts/requirements.txt
    env python3.9 -m pip install /opt/ottopia/keepalived-exporter/ottopia_logging-0.1.1*.whl

    return $?
}

main
