using System.Collections.Generic;
using Pulumi;
using AzureNative = Pulumi.AzureNative;
using SyncedFolder = Pulumi.SyncedFolder;

return await Deployment.RunAsync(() =>
{
    var config = new Config();
    var path = config.Get("path") ?? "./www";
    var indexDocument = config.Get("indexDocument") ?? "index.html";
    var errorDocument = config.Get("errorDocument") ?? "error.html";
    var resourceGroup = new AzureNative.Resources.ResourceGroup("resource-group");

    var account = new AzureNative.Storage.StorageAccount("account", new()
    {
        ResourceGroupName = resourceGroup.Name,
        Kind = "StorageV2",
        Sku = new AzureNative.Storage.Inputs.SkuArgs
        {
            Name = "Standard_LRS",
        },
    });

    var container = new AzureNative.Storage.BlobContainer("container", new()
    {
        AccountName = account.Name,
        ResourceGroupName = resourceGroup.Name,
        PublicAccess = AzureNative.Storage.PublicAccess.None,
    });

    var blobSAS = AzureNative.Storage.ListStorageAccountServiceSAS.Invoke(new()
    {
        ResourceGroupName = resourceGroup.Name,
        AccountName = account.Name,
        Protocols = AzureNative.Storage.HttpProtocol.Https,
        SharedAccessStartTime = "2022-01-01",
        SharedAccessExpiryTime = "2030-01-01",
        Resource = "c",
        Permissions = "r",
        CanonicalizedResource = Output.Tuple(account.Name, container.Name).Apply(values => $"/blob/{values.Item1}/{values.Item2}"),
        ContentType = "application/json",
        CacheControl = "max-age=5",
        ContentDisposition = "inline",
        ContentEncoding = "deflate",
    });

    var source = new FileArchive("./api");

    var website = new AzureNative.Storage.StorageAccountStaticWebsite("website", new()
    {
        AccountName = account.Name,
        ResourceGroupName = resourceGroup.Name,
        IndexDocument = indexDocument,
        Error404Document = errorDocument,
    });

    var syncedFolder = new SyncedFolder.AzureBlobFolder("synced-folder", new()
    {
        Path = path,
        ResourceGroupName = resourceGroup.Name,
        StorageAccountName = account.Name,
        ContainerName = website.ContainerName,
    });

    var plan = new AzureNative.Web.AppServicePlan("plan", new()
    {
        ResourceGroupName = resourceGroup.Name,
        Sku = new AzureNative.Web.Inputs.SkuDescriptionArgs
        {
            Name = "Y1",
            Tier = "Dynamic",
        },
    });

    var blob = new AzureNative.Storage.Blob("blob", new()
    {
        AccountName = account.Name,
        ResourceGroupName = resourceGroup.Name,
        ContainerName = container.Name,
        Source = source,
    });

    var app = new AzureNative.Web.WebApp("app", new()
    {
        ResourceGroupName = resourceGroup.Name,
        ServerFarmId = plan.Id,
        Kind = "FunctionApp",
        SiteConfig = new AzureNative.Web.Inputs.SiteConfigArgs
        {
            AppSettings = new[]
            {
                new AzureNative.Web.Inputs.NameValuePairArgs
                {
                    Name = "FUNCTIONS_WORKER_RUNTIME",
                    Value = "node",
                },
                new AzureNative.Web.Inputs.NameValuePairArgs
                {
                    Name = "WEBSITE_NODE_DEFAULT_VERSION",
                    Value = "~14",
                },
                new AzureNative.Web.Inputs.NameValuePairArgs
                {
                    Name = "FUNCTIONS_EXTENSION_VERSION",
                    Value = "~3",
                },
                new AzureNative.Web.Inputs.NameValuePairArgs
                {
                    Name = "WEBSITE_RUN_FROM_PACKAGE",
                    Value = Output.Tuple(account.Name, container.Name, blob.Name, blobSAS.Apply(result => result.ServiceSasToken)).Apply(values =>
                    {
                        var accountName = values.Item1;
                        var containerName = values.Item2;
                        var blobName = values.Item3;
                        var token = values.Item4;
                        var url = $"https://{accountName}.blob.core.windows.net/{containerName}/{blobName}?{token}";
                        return url;
                    }),
                },
            },
            Cors = new AzureNative.Web.Inputs.CorsSettingsArgs
            {
                AllowedOrigins = new[]
                {
                    "*",
                },
            },
        },
    });

    var cfg = app.DefaultHostName.Apply(hostname => new Dictionary<string, string>() { ["apiEndpoint"] = $"https://{hostname}/api" });
    var siteConfig = new AzureNative.Storage.Blob("config.json", new()
    {
        // Source = app.DefaultHostName.Apply(hostname => new Pulumi.StringAsset(JsonSerializer.Serialize(cfg))),
        Source = app.DefaultHostName.Apply(hostname => new Pulumi.StringAsset("{ \"api\": \"https://" + hostname  + "/api\" }") as AssetOrArchive),
        ContentType = "application/json",
        AccountName = account.Name,
        ResourceGroupName = resourceGroup.Name,
        ContainerName = website.ContainerName,
    });

    return new Dictionary<string, object?>
    {
        ["originURL"] = account.PrimaryEndpoints.Apply(primaryEndpoints => primaryEndpoints.Web),
        ["originHostname"] = account.PrimaryEndpoints.Apply(primaryEndpoints => primaryEndpoints.Web),
        ["apiURL"] = app.DefaultHostName.Apply(defaultHostName => $"https://{defaultHostName}/api"),
        ["apiHostname"] = app.DefaultHostName,
    };
});
