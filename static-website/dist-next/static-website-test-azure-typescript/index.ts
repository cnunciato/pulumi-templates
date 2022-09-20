import * as pulumi from "@pulumi/pulumi";
import * as azure_native from "@pulumi/azure-native";
import * as synced_folder from "@pulumi/synced-folder";
import * as cdn from "@pulumi/azure-native/cdn";
import * as network from "@pulumi/azure-native/network";
import * as resources from "@pulumi/azure-native/resources";

// Import the program's configuration settings.
const config = new pulumi.Config();
const path = config.get("path") || "./www";
const indexDocument = config.get("indexDocument") || "index.html";
const errorDocument = config.get("errorDocument") || "error.html";
const domain = config.require("domain");
const subdomain = config.require("subdomain");
const zoneResourceGroupName = config.require("zoneResourceGroupName");
const domainName = `${subdomain}.${domain}`;

// Create a resource group for the website.
const resourceGroup = new azure_native.resources.ResourceGroup("resource-group", {});

// Create a blob storage account.
const account = new azure_native.storage.StorageAccount("account", {
    resourceGroupName: resourceGroup.name,
    kind: "StorageV2",
    sku: {
        name: "Standard_LRS",
    },
});

// Configure the storage account as a website.
const website = new azure_native.storage.StorageAccountStaticWebsite("website", {
    resourceGroupName: resourceGroup.name,
    accountName: account.name,
    indexDocument: indexDocument,
    error404Document: errorDocument,
});

// Use a synced folder to manage the files of the website.
const syncedFolder = new synced_folder.AzureBlobFolder("synced-folder", {
    path: path,
    resourceGroupName: resourceGroup.name,
    storageAccountName: account.name,
    containerName: website.containerName,
});

// Create a CDN profile.
const profile = new azure_native.cdn.Profile("profile", {
    resourceGroupName: resourceGroup.name,
    sku: {
        name: "Standard_Microsoft",
    },
});

// Pull the hostname out of the storage-account endpoint.
const originHostname = account.primaryEndpoints.apply(endpoints => new URL(endpoints.web)).hostname;

// Create a CDN endpoint to distribute and cache the website.
const endpoint = new azure_native.cdn.Endpoint("endpoint", {
    resourceGroupName: resourceGroup.name,
    profileName: profile.name,
    isHttpAllowed: true,
    isHttpsAllowed: true,
    isCompressionEnabled: true,
    contentTypesToCompress: [
        "text/html",
        "text/css",
        "application/javascript",
        "application/json",
        "image/svg+xml",
        "font/woff",
        "font/woff2",
    ],
    originHostHeader: originHostname,
    origins: [{
        name: account.name,
        hostName: originHostname,
    }],
});

const dnsResourceGroup = resources.getResourceGroupOutput({
    resourceGroupName: zoneResourceGroupName
});

const cname = new network.RecordSet("cname", {
    resourceGroupName: dnsResourceGroup.name,
    relativeRecordSetName: subdomain,
    zoneName: domain,
    recordType: "CNAME",
    targetResource: {
        id: endpoint.id,
    },
});

// Create a custom domain.
const customDomain = new cdn.CustomDomain("domain", {
    resourceGroupName: resourceGroup.name,
    profileName: profile.name,
    endpointName: endpoint.name,
    hostName: cname.fqdn.apply(s => s.split(".").filter(s => s !== "").join(".")),
});

// const cert = new local.Command("enable-https", {
//     create: pulumi.interpolate`az cdn custom-domain enable-https --resource-group ${resourceGroup.name} --profile-name ${profile.name} --endpoint-name ${endpoint.name} --name ${domain.name}`,
// });

// Export the URLs and hostnames of the storage account and CDN.
export const originURL = pulumi.interpolate`http://${originHostname}`;
export { originHostname };
export const cdnURL = pulumi.interpolate`https://${endpoint.hostName}`;
export const cdnHostname = endpoint.hostName;
export const domainURL = `http://${domainName}`;
