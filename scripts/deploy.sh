#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

for cloud in $CLOUDS; do
    for lang in $LANGUAGES; do
        pulumi -C "$cloud/$lang" stack init dev || pulumi -C "$cloud/$lang" stack select dev
        pulumi -C "$cloud/$lang" config set aws:region us-west-2
        pulumi -C "$cloud/$lang" config set azure-native:location WestUS
        pulumi -C "$cloud/$lang" config set gcp:project pulumi-development
        pulumi -C "$cloud/$lang" config set path ./site
        pulumi -C "$cloud/$lang" config set indexDocument index.html
        pulumi -C "$cloud/$lang" config set errorDocument error.html
        pulumi -C "$cloud/$lang" up --yes
    done
done
