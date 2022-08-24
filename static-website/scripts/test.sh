#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

for cloud in $CLOUDS; do
    for lang in $LANGUAGES; do
        test_dir="test/static-website-test-${cloud}-${lang}"

        echo "##"
        echo "# Testing $test_dir"
        echo "##"

        pulumi -C "$test_dir" destroy --yes || true
        pulumi -C "$test_dir" stack rm --yes || true
        rm -rf "$test_dir" && mkdir -p "$test_dir"

        pushd "$test_dir"
            pulumi new --yes "../../dist/static-website-${cloud}-${lang}"
            pulumi up --yes
            pulumi destroy --yes
            pulumi stack rm --yes
        popd
    done
done
