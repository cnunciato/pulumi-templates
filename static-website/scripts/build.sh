#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

mkdir -p dist

for cloud in $CLOUDS; do
    pushd "$cloud"
        for lang in $LANGUAGES; do

            # Create a Pulumi.yaml to generate projects from.
            cat "header-project.yaml" "append.yaml" | sed -e "s/{cloud}/$cloud/g" -e "s/{lang}/$lang/g" > Pulumi.yaml

            # Convert the program into the current language.
            rm -rf "$lang"
            pulumi convert --out "$lang" --language "$lang"

            # Copy the site folder into the project.
            cp -R "../site" "$lang/"

            # Prepare and copy the completed template to the dist folder.
            template_dir="../dist/static-website-${cloud}-${lang}"
            cp -R "$lang" "$template_dir"
            cat "header-template.yaml" "template.yaml" "append.yaml" > "$template_dir/Pulumi.yaml"
            cat "append.yaml" > "$template_dir/Pulumi.yaml.append"

            # Remove the generated Pulumi.yaml.
            rm -f Pulumi.yaml
        done
    popd

    # Fixups.
    sed -i '' 's/"github.com\/pulumi\/pulumi-synced-folder\/sdk\/go\/synced-folder"/synced "github.com\/pulumi\/pulumi-synced-folder\/sdk\/go\/synced-folder"/g' "$cloud/go/main.go" || true
    sed -i '' 's/synced - folder/\synced/g' "$cloud/go/main.go" || true
    sed -i '' 's/\&synced-folder/\&synced/g' "$cloud/go/main.go" || true
done

# Pull the junk out of the dist folder.
find dist -name 'node_modules' -type d -prune -exec rm -rf '{}' +
find dist -name 'bin' -type d -prune -exec rm -rf '{}' +
find dist -name 'obj' -type d -prune -exec rm -rf '{}' +
find dist -name 'lib' -type d -prune -exec rm -rf '{}' +
find dist -name 'include' -type d -prune -exec rm -rf '{}' +
find dist -name 'pyvenv.cfv' -prune -exec rm '{}' +
find dist -name 'yarn.lock' -prune -exec rm '{}' +
find dist -name 'go.sum' -prune -exec rm '{}' +
