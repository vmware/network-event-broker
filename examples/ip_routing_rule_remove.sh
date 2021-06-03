#!/bin/bash

LINK=$(printenv LINK)
DEV="ens37"
TABLE="10"

if [ "$LINK" != "$DEV" ]; then
    exit 0
fi

SOURCE_IP=$(ip route show table main | grep $DEV | grep default | cut -d ' ' -f 9)

echo "Removing ip rules for $DEV Source IP=$SOURCE_IP Table=$TABLE"

ip rule delete from all to $SOURCE_IP lookup $TABLE || true
ip rule delete from $SOURCE_IP lookup $TABLE || true
