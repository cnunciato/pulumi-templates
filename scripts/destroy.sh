#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

for cloud in $CLOUDS; do
    for lang in $LANGUAGES; do
        if [ -d "$cloud/$lang" ]; then
            pulumi -C "$cloud/$lang" stack select dev && \
            pulumi -C "$cloud/$lang" destroy --yes && \
            pulumi -C "$cloud/$lang" stack rm --yes || true
        fi
    done
    rm -f "$cloud/Pulumi.yaml"
done
