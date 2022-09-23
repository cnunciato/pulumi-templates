import * as pulumi from "@pulumi/pulumi";
import * as azure_native from "@pulumi/azure-native";
import * as synced_folder from "@pulumi/synced-folder";

const config = new pulumi.Config();
const path = config.get("path") || "./www";
const indexDocument = config.get("indexDocument") || "index.html";
const errorDocument = config.get("errorDocument") || "error.html";

const resourceGroup = new azure_native.resources.ResourceGroup("resource-group", {});

const account = new azure_native.storage.StorageAccount("account", {
    resourceGroupName: resourceGroup.name,
    kind: "StorageV2",
    sku: {
        name: "Standard_LRS",
    },
});

const container = new azure_native.storage.BlobContainer("container", {
    accountName: account.name,
    resourceGroupName: resourceGroup.name,
    publicAccess: azure_native.storage.PublicAccess.None,
});

const blobSAS = azure_native.storage.listStorageAccountServiceSASOutput({
    resourceGroupName: resourceGroup.name,
    accountName: account.name,
    protocols: azure_native.storage.HttpProtocol.Https,
    sharedAccessStartTime: "2022-01-01",
    sharedAccessExpiryTime: "2030-01-01",
    resource: "c",
    permissions: "r",
    canonicalizedResource: pulumi.interpolate`/blob/${account.name}/${container.name}`,
    contentType: "application/json",
    cacheControl: "max-age=5",
    contentDisposition: "inline",
    contentEncoding: "deflate",
});

const website = new azure_native.storage.StorageAccountStaticWebsite("website", {
    accountName: account.name,
    resourceGroupName: resourceGroup.name,
    indexDocument: indexDocument,
    error404Document: errorDocument,
});

const syncedFolder = new synced_folder.AzureBlobFolder("synced-folder", {
    path: path,
    resourceGroupName: resourceGroup.name,
    storageAccountName: account.name,
    containerName: website.containerName,
});

const plan = new azure_native.web.AppServicePlan("plan", {
    resourceGroupName: resourceGroup.name,
    sku: {
        name: "Y1",
        tier: "Dynamic",
    },
});

const blob = new azure_native.storage.Blob("blob", {
    accountName: account.name,
    resourceGroupName: resourceGroup.name,
    containerName: container.name,
    source: new pulumi.asset.FileArchive("./api"),
});

const functionApp = new azure_native.web.WebApp("function-app", {
    resourceGroupName: resourceGroup.name,
    serverFarmId: plan.id,
    kind: "FunctionApp",
    siteConfig: {
        appSettings: [
            {
                name: "runtime",
                value: "node",
            },
            {
                name: "FUNCTIONS_WORKER_RUNTIME",
                value: "node",
            },
            {
                name: "WEBSITE_RUN_FROM_PACKAGE",
                value: pulumi.all([account.name, container.name, blob.name, blobSAS]).apply(([accountName, containerName, blobName, blobSAS]) => `https://${accountName}.blob.core.windows.net/${containerName}/${blobName}?${blobSAS.serviceSasToken}`),
            },
            {
                name: "WEBSITE_NODE_DEFAULT_VERSION",
                value: "~14",
            },
            {
                name: "FUNCTIONS_EXTENSION_VERSION",
                value: "~3",
            },
        ],
        cors: {
            allowedOrigins: ["*"],
        },
    },
});

const configFile = new azure_native.storage.Blob("config.json", {
    source: functionApp.defaultHostName.apply(host => new pulumi.asset.StringAsset(JSON.stringify({ api: `https://${host}/api` }))),
    contentType: "application/json",
    accountName: account.name,
    resourceGroupName: resourceGroup.name,
    containerName: website.containerName,
});

export const originURL = account.primaryEndpoints.apply(primaryEndpoints => primaryEndpoints.web);
export const originHostname = account.primaryEndpoints.apply(primaryEndpoints => primaryEndpoints.web);
export const apiURL = pulumi.interpolate`https://${functionApp.defaultHostName}/api`;
export const apiHostname = functionApp.defaultHostName;
