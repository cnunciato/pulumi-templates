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
		source := pulumi.NewFileArchive("./src")
		website, err := storage.NewStorageAccountStaticWebsite(ctx, "website", &storage.StorageAccountStaticWebsiteArgs{
			ResourceGroupName: resourceGroup.Name,
			AccountName:       account.Name,
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
			Source:            Archive(source),
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
						Name:  pulumi.String("runtime"),
						Value: pulumi.String("node"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("FUNCTIONS_WORKER_RUNTIME"),
						Value: pulumi.String("node"),
					},
					&web.NameValuePairArgs{
						Name: pulumi.String("WEBSITE_RUN_FROM_PACKAGE"),
						Value: pulumi.All(account.Name, container.Name, blob.Name, blobSAS).ApplyT(func(_args []interface{}) (string, error) {
							accountName := _args[0].(string)
							containerName := _args[1].(string)
							blobName := _args[2].(string)
							blobSAS := _args[3].(storage.ListStorageAccountServiceSASResult)
							return fmt.Sprintf("https://%v.blob.core.windows.net/%v/%v?%v", accountName, containerName, blobName, blobSAS.ServiceSasToken), nil
						}).(pulumi.StringOutput),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("WEBSITE_NODE_DEFAULT_VERSION"),
						Value: pulumi.String("~12"),
					},
					&web.NameValuePairArgs{
						Name:  pulumi.String("FUNCTIONS_EXTENSION_VERSION"),
						Value: pulumi.String("~3"),
					},
				},
			},
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
			return fmt.Sprintf("https://%v/api/hello-world?name=Pulumi", defaultHostName), nil
		}).(pulumi.StringOutput))
		return nil
	})
}
