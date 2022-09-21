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
const blobSAS = pulumi.all([account.name, resourceGroup.name, account.name, container.name]).apply(([accountName, resourceGroupName, accountName1, containerName]) => azure_native.storage.listStorageAccountServiceSASOutput({
    accountName: accountName,
    protocols: azure_native.storage.HttpProtocol.Https,
    sharedAccessExpiryTime: "2030-01-01",
    sharedAccessStartTime: "2021-01-01",
    resourceGroupName: resourceGroupName,
    resource: "c",
    permissions: "r",
    canonicalizedResource: `/blob/${accountName1}/${containerName}`,
    contentType: "application/json",
    cacheControl: "max-age=5",
    contentDisposition: "inline",
    contentEncoding: "deflate",
}));
const source = new pulumi.asset.FileArchive("./api");
const website = new azure_native.storage.StorageAccountStaticWebsite("website", {
    resourceGroupName: resourceGroup.name,
    accountName: account.name,
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
    source: source,
});
const app = new azure_native.web.WebApp("app", {
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
                value: "~12",
            },
            {
                name: "FUNCTIONS_EXTENSION_VERSION",
                value: "~3",
            },
        ],
    },
});
export const originURL = account.primaryEndpoints.apply(primaryEndpoints => primaryEndpoints.web);
export const originHostname = account.primaryEndpoints.apply(primaryEndpoints => primaryEndpoints.web);
export const apiURL = pulumi.interpolate`https://${app.defaultHostName}/api/hello-world?name=Pulumi`;
