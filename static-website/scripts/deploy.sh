#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

for cloud in $CLOUDS; do
    for lang in $LANGUAGES; do
        pushd "$cloud/$lang"
            pulumi stack init dev || pulumi stack select dev

            if [ "$cloud" == "aws" ]; then
                pulumi config set aws:region us-west-2
            fi

            if [ "$cloud" == "azure" ]; then
                pulumi config set azure-native:location WestUS
            fi

            if [ "$cloud" == "gcp" ]; then
                pulumi config set gcp:project pulumi-development
            fi

            if [ "$lang" == "python" ]; then
                source bin/activate
            fi

            pulumi config set path ./site
            pulumi config set indexDocument index.html
            pulumi config set errorDocument error.html
            pulumi up --yes
        popd
    done
done
