#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

cloud=""
lang=""

read -p "cloud-lang: " cloud_lang

test_dir="test/static-website-test-${cloud_lang}"
echo "##"
echo "# Testing $test_dir"
echo "##"

pulumi -C "$test_dir" destroy --yes || true
pulumi -C "$test_dir" stack rm --yes || true
rm -rf "$test_dir" && mkdir -p "$test_dir"

pushd "$test_dir"
    pulumi new "../../dist/static-website-${cloud_lang}"
    pulumi up
    pulumi destroy
    pulumi stack rm
popd
