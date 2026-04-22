#!/bin/bash

# default config file path
conf="${WGUI_CONFIG_FILE_PATH:-/etc/wireguard/wg0.conf}"

# manage wireguard stop/start with the container
case $WGUI_MANAGE_START in (1|t|T|true|True|TRUE)
    wg-quick up "$conf"
    trap 'wg-quick down "$conf"' SIGTERM # catches container stop
esac

# manage wireguard restarts
case $WGUI_MANAGE_RESTART in (1|t|T|true|True|TRUE)
    [[ -f $conf ]] || touch "$conf" # inotifyd needs file to exist
    inotifyd - "$conf":w | while read -r event file; do
        wg-quick down "$file"
        wg-quick up "$file"
    done &
esac

./wg-ui &
wait $!
