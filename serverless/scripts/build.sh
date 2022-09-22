#!/bin/bash

set -o errexit -o pipefail
source ./scripts/common.sh

rm -rf dist && mkdir dist

for cloud in $CLOUDS; do
    pushd "$cloud"
        for lang in $LANGUAGES; do

            # Create a Pulumi.yaml to generate projects from.
            cat "header-project.yaml" "body.yaml" | sed -e "s/{cloud}/$cloud/g" -e "s/{lang}/$lang/g" > Pulumi.yaml

            # Convert the program into the current language.
            rm -rf "$lang"
            pulumi convert --out "$lang" --language "$lang" || true

            # Do some post-`pulumi convert` fixups.
            if [ "$lang" == "go" ]; then
                sed -i '' 's/"github.com\/pulumi\/pulumi-synced-folder\/sdk\/go\/synced-folder"/synced "github.com\/pulumi\/pulumi-synced-folder\/sdk\/go\/synced-folder"/g' "$lang/main.go" || true
                sed -i '' 's/synced - folder/\synced/g' "$lang/main.go" || true
                sed -i '' 's/\&synced-folder/\&synced/g' "$lang/main.go" || true
            fi

            if [ "$lang" == "yaml" ]; then
                sed -i '' 's/type: string/type: String/g' "$lang/Main.yaml" || true
            fi

            # Copy the www folder into the project.
            cp -R "../www" "$lang/"
            cp -R "../api" "$lang/"

            # Prepare and copy the completed template to the dist folder.
            template_dir="../dist/serverless-${cloud}-${lang}"
            cp -R "$lang" "$template_dir"
            cat "header-template.yaml" "template.yaml" > "$template_dir/Pulumi.yaml"

            if [ "$lang" == "yaml" ]; then
                cat "body.yaml" >> "$template_dir/Pulumi.yaml"
            fi

            # Set the appropriate runtime for the template.
            if [ "$lang" == "typescript" ] || [ "$lang" == "javascript" ]; then
                runtime="nodejs"
            elif [ "$lang" == "csharp" ] || [ "$lang" == "fsharp" ] || [ "$lang" == "visualbasic" ]; then
                runtime="dotnet"
            else
                runtime="$lang"
            fi
            sed -i '' "s/{runtime}/${runtime}/g" "$template_dir/Pulumi.yaml" || true

            # Copy in the .append file for YAML templates.
            if [ "$lang" == "yaml" ]; then
                cat "body.yaml" > "$template_dir/Pulumi.yaml.append"
            fi

            # Delete the Main.yamls -- I don't think we need them.
            if [ "$lang" == "yaml" ]; then
                rm -f "$template_dir/Main.yaml"
            fi

            # Remove the generated Pulumi.yaml.
            rm -f Pulumi.yaml
        done
    popd
done

# Pull the junk out of the dist folder.
find dist -name 'node_modules' -type d -prune -exec rm -rf '{}' +
find dist -name 'bin' -type d -prune -exec rm -rf '{}' +
find dist -name 'obj' -type d -prune -exec rm -rf '{}' +
find dist -name 'lib' -type d -prune -exec rm -rf '{}' +
find dist -name 'include' -type d -prune -exec rm -rf '{}' +
find dist -name 'pyvenv.cfg' -prune -exec rm '{}' +
find dist -name 'yarn.lock' -prune -exec rm '{}' +
find dist -name 'go.sum' -prune -exec rm '{}' +
