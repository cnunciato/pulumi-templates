#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

templates_repo="$HOME/go/src/github.com/pulumi/templates"
echo "$templates_repo/static-website-*"

rm -rf $templates_repo/static-website-*
cp -R dist/* $templates_repo/

code $templates_repo
