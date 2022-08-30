# pulumi-templates-gen

A bunch of YAML and Bash that I use for producing Pulumi project templates.

```bash
nvm use 16 && gvm use 1.17
make build
```

## Manual `pulumi-convert` fixups

The `scripts/build.sh` file contains a handful of fixes, but many are beyond my Bash-scripting/patience level. Several changes will still need to be made to the rendered templates in `dist` after running `make build`.

## Testing rendered templates

After you've made any manual fixups, you can run `make test` to test all of the templates in `dist` end to end, or `make test-one` to test just one of them. Once all of the tests finish successfully, and you've made whatever aesthetic changes you want to make to the code, you can run `make copy` to copy everything into your local clone of `pulumi/templates` and PR from there.
