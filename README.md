# pulumi-templates-gen

A bunch of YAML and Bash that I use for producing Pulumi project templates.

```bash
nvm use 16
gvm use 1.17
make build
make clean
```

## Manual `pulumi-convert` fixups

The `scripts/build.sh` file contains a handful of fixes, but some are beyond my Bash-scripting/patience level. The following changes still need to be made manually to the rendered templates in `dist` after running `make build`.

### static-website

#### azure

Stemming from https://github.com/pulumi/pulumi-yaml/issues/317:

1. The CDN endpoint needs a hostname, not a fully qualified URL.

    In `dist/static-website-azure-yaml/Pulumi.yaml` and `Pulumi.yaml.append`, add the following `variables` block:

    ```yaml
    variables:
      originHostname:
        Fn::Select:
          - 2
          - Fn::Split:
            - /
            - ${account.primaryEndpoints.web}

    # ...

    originHostHeader: ${originHostname}
    origins:
      - name: ${account.name}
        hostName: ${originHostname}
    ```

2. Similarly, in `dist/static-website-azure-typescript/index.ts`, make the following changes:

    ```typescript
    const originHostname = account.primaryEndpoints.apply(endpoints => new URL(endpoints.web)).hostname;

    // ...

    originHostHeader: originHostname,
    origins: [{
        name: account.name,
        hostName: originHostname,
    }],
    ```

3. And in `dist/static-website-azure-go/main.go`:

    ```go
    originHostname := account.PrimaryEndpoints.ApplyT(func(endpoints storage.EndpointsResponse) (string, error) {
        parsed, err := url.Parse(endpoints.Web)
        if err != nil {
            return "", err
        }
        return parsed.Hostname(), nil
    }).(pulumi.StringOutput)

    // ...

    OriginHostHeader: originHostname,
    Origins: cdn.DeepCreatedOriginArray{
        &cdn.DeepCreatedOriginArgs{
            Name: account.Name,
            HostName: originHostname,
        },
    },
    ```

4. And in `dist/static-website-azure-python/__main__.py`:

    ```python
    import urllib

    #...

    origin_hostname = account.primary_endpoints.web.apply(lambda endpoint: urllib.parse.urlparse(endpoint).hostname)

    # ...

    origin_host_header=origin_hostname,
    origins=[azure_native.cdn.DeepCreatedOriginArgs(
        name=account.name,
        host_name=origin_hostname,
    )])
    ```

5. And then finally, in `dist/static-website-azure-csharp/Program.cs`:

    ```csharp
    using System;

    // ...

    var originHostname = account.PrimaryEndpoints.Apply(endpoints => new Uri(endpoints.Web).Host);

    // ...

    OriginHostHeader = originHostname,
    Origins = new[]
    {
        new AzureNative.Cdn.Inputs.DeepCreatedOriginArgs
        {
            Name = account.Name,
            HostName = originHostname,
        },
    },
    ```

Once you've made these changes, you can run `make test` to test the  templates in `dist`, make aesthetic changes to the code, add comments, etc., and finally `make copy` to copy everything over to pulumi/templates.
