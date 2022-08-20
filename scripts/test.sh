#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

cloud=""
lang=""

read -p "cloud: " cloud
read -p "lang: " lang

test_dir="test/static-website-test-${cloud}-${lang}"
rm -rf "test/*" && mkdir -p "$test_dir"

pulumi -C $test_dir new "../../dist/static-website-${cloud}-${lang}"
pulumi -C $test_dir up
pulumi -C $test_dir destroy
pulumi -C $test_dir stack rm
