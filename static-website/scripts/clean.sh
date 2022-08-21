#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

rm -rf dist test

for cloud in $CLOUDS; do
    for lang in $LANGUAGES; do
        rm -rf $cloud/$lang "$cloud/Pulumi.yaml"
    done
done
