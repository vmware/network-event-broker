#!/bin/bash

LINK=$(printenv LINK)
DEV="ens37"
TABLE="10"

if [ "$LINK" != "$DEV" ]; then
    exit 0
fi

SOURCE_IP=$(ip route show table main | grep $DEV | grep default | cut -d ' ' -f 9)

echo "Configuring ip rules for $DEV Source IP=$SOURCE_IP Table=$TABLE"

ip rule add from $SOURCE_IP table $TABLE
ip rule add to $SOURCE_IP table $TABLE
