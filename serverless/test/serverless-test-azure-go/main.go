package main

import (
	"fmt"

	resources "github.com/pulumi/pulumi-azure-native/sdk/go/azure/resources"
	storage "github.com/pulumi/pulumi-azure-native/sdk/go/azure/storage"
	web "github.com/pulumi/pulumi-azure-native/sdk/go/azure/web"
	synced "github.com/pulumi/pulumi-synced-folder/sdk/go/synced-folder"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		path := "./www"
		if param := cfg.Get("path"); param != "" {
			path = param
		}
		indexDocument := "index.html"
		if param := cfg.Get("indexDocument"); param != "" {
			indexDocument = param
		}
		errorDocument := "error.html"
		if param := cfg.Get("errorDocument"); param != "" {
			errorDocument = param
		}
		resourceGroup, err := resources.NewResourceGroup(ctx, "resource-group", nil)
		if err != nil {
			return err
		}
		account, err := storage.NewStorageAccount(ctx, "account", &storage.StorageAccountArgs{
			ResourceGroupName: resourceGroup.Name,
			Kind:              pulumi.String("StorageV2"),
			Sku: &storage.SkuArgs{
				Name: pulumi.String("Standard_LRS"),
			},
		})
		if err != nil {
			return err
		}
		container, err := storage.NewBlobContainer(ctx, "container", &storage.BlobContainerArgs{
			AccountName:       account.Name,
			ResourceGroupName: resourceGroup.Name,
			PublicAccess:      storage.PublicAccessNone,
		})
		if err != nil {
			return err
		}

		website, err := storage.NewStorageAccountStaticWebsite(ctx, "website", &storage.StorageAccountStaticWebsiteArgs{
			AccountName:       account.Name,
			ResourceGroupName: resourceGroup.Name,
			IndexDocument:     pulumi.String(indexDocument),
			Error404Document:  pulumi.String(errorDocument),
		})
		if err != nil {
			return err
		}
		_, err = synced.NewAzureBlobFolder(ctx, "synced-folder", &synced.AzureBlobFolderArgs{
			Path:               pulumi.String(path),
			ResourceGroupName:  resourceGroup.Name,
			StorageAccountName: account.Name,
			ContainerName:      website.ContainerName,
		})
		if err != nil {
			return err
		}
		plan, err := web.NewAppServicePlan(ctx, "plan", &web.AppServicePlanArgs{
			ResourceGroupName: resourceGroup.Name,
			Sku: &web.SkuDescriptionArgs{
				Name: pulumi.String("Y1"),
				Tier: pulumi.String("Dynamic"),
			},
		})
		if err != nil {
			return err
		}
		blob, err := storage.NewBlob(ctx, "blob", &storage.BlobArgs{
			AccountName:       account.Name,
			ResourceGroupName: resourceGroup.Name,
			ContainerName:     container.Name,
			Source:            pulumi.NewFileArchive("./api"),
		})
		if err != nil {
			return err
		}

		app, err := web.NewWebApp(ctx, "app", &web.WebAppArgs{
			ResourceGroupName: resourceGroup.Name,
			ServerFarmId:      plan.ID(),
			Kind:              pulumi.String("FunctionApp"),
			SiteConfig: &web.SiteConfigArgs{
				AppSettings: web.NameValuePairArray{
					&web.NameValuePairArgs{
						Name:  pulumi.String("FUNCTIONS_WORKER_RUNTIME"),
						Value: pulumi.String("node"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("WEBSITE_NODE_DEFAULT_VERSION"),
						Value: pulumi.String("~14"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("FUNCTIONS_EXTENSION_VERSION"),
						Value: pulumi.String("~3"),
					},
					&web.NameValuePairArgs{
						Name: pulumi.String("WEBSITE_RUN_FROM_PACKAGE"),
						Value: pulumi.All(resourceGroup.Name, account.Name, container.Name, blob.Name).ApplyT(
							func(args []interface{}) string {
								ctx.Log.Info("Testing", nil)

								resourceGroupName := args[0].(string)
								accountName := args[1].(string)
								containerName := args[2].(string)
								blobName := args[3].(string)

								protocol := storage.HttpProtocolHttps
								result, err := storage.ListStorageAccountServiceSAS(ctx, &storage.ListStorageAccountServiceSASArgs{
									ResourceGroupName:      resourceGroupName,
									AccountName:            accountName,
									Protocols:              &protocol,
									SharedAccessStartTime:  pulumi.StringRef("2022-01-01"),
									SharedAccessExpiryTime: pulumi.StringRef("2030-01-01"),
									Resource:               pulumi.StringRef("c"),
									Permissions:            pulumi.StringRef("r"),
									ContentType:            pulumi.StringRef("application/json"),
									CacheControl:           pulumi.StringRef("max-age=5"),
									ContentDisposition:     pulumi.StringRef("inline"),
									ContentEncoding:        pulumi.StringRef("deflate"),
									CanonicalizedResource:  fmt.Sprintf("/blob/%s/%s", accountName, containerName),
								})
								if err != nil {
									ctx.Log.Info(err.Error(), nil)
									return ""
								}

								token := result.ServiceSasToken
								url := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s", accountName, containerName, blobName, token)

								ctx.Log.Info(url, nil)

								return url
							}).(pulumi.StringPtrInput),
					},
				},
				Cors: &web.CorsSettingsArgs{
					AllowedOrigins: pulumi.StringArray{
						pulumi.String("*"),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = storage.NewBlob(ctx, "config.json", &storage.BlobArgs{
			AccountName:       account.Name,
			ResourceGroupName: resourceGroup.Name,
			ContainerName:     website.ContainerName,
			ContentType:       pulumi.StringPtr("application/json"),
			Source: app.DefaultHostName.ApplyT(func(hostname string) pulumi.AssetOrArchiveOutput {
				json := fmt.Sprintf("{ \"api\": \"https://%s/api\" }", hostname)
				return pulumi.NewStringAsset(json).ToAssetOrArchiveOutput()
			}).(pulumi.AssetOrArchiveOutput),
		})
		if err != nil {
			return err
		}

		ctx.Export("originURL", account.PrimaryEndpoints.ApplyT(func(primaryEndpoints storage.EndpointsResponse) (string, error) {
			return primaryEndpoints.Web, nil
		}).(pulumi.StringOutput))
		ctx.Export("originHostname", account.PrimaryEndpoints.ApplyT(func(primaryEndpoints storage.EndpointsResponse) (string, error) {
			return primaryEndpoints.Web, nil
		}).(pulumi.StringOutput))
		ctx.Export("apiURL", app.DefaultHostName.ApplyT(func(defaultHostName string) (string, error) {
			return fmt.Sprintf("https://%v/api", defaultHostName), nil
		}).(pulumi.StringOutput))
		ctx.Export("apiHostname", app.DefaultHostName)
		return nil
	})
}
