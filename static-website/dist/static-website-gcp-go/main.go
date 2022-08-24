package main

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	synced "github.com/pulumi/pulumi-synced-folder/sdk/go/synced-folder"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		gcpProject := "pulumi-development"
		if param := cfg.Get("gcpProject"); param != "" {
			gcpProject = param
		}
		path := "./site"
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
		bucket, err := storage.NewBucket(ctx, "bucket", &storage.BucketArgs{
			Location: pulumi.String("US"),
			Website: &storage.BucketWebsiteArgs{
				MainPageSuffix: pulumi.String(indexDocument),
				NotFoundPage:   pulumi.String(errorDocument),
			},
		})
		if err != nil {
			return err
		}
		_, err = storage.NewBucketIAMBinding(ctx, "bucket-iam-binding", &storage.BucketIAMBindingArgs{
			Bucket: bucket.Name,
			Role:   pulumi.String("roles/storage.objectViewer"),
			Members: pulumi.StringArray{
				pulumi.String("allUsers"),
			},
		})
		if err != nil {
			return err
		}
		_, err = synced.NewGoogleCloudFolder(ctx, "synced-folder", &synced.GoogleCloudFolderArgs{
			Path:       pulumi.String(path),
			BucketName: bucket.Name,
		})
		if err != nil {
			return err
		}
		backendBucket, err := compute.NewBackendBucket(ctx, "backend-bucket", &compute.BackendBucketArgs{
			BucketName: bucket.Name,
			EnableCdn:  pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		ip, err := compute.NewGlobalAddress(ctx, "ip", nil)
		if err != nil {
			return err
		}
		urlMap, err := compute.NewURLMap(ctx, "url-map", &compute.URLMapArgs{
			DefaultService: backendBucket.SelfLink,
		})
		if err != nil {
			return err
		}
		httpProxy, err := compute.NewTargetHttpProxy(ctx, "http-proxy", &compute.TargetHttpProxyArgs{
			UrlMap: urlMap.SelfLink,
		})
		if err != nil {
			return err
		}
		_, err = compute.NewGlobalForwardingRule(ctx, "http-forwarding-rule", &compute.GlobalForwardingRuleArgs{
			IpAddress:  pulumi.String("ip.address"),
			IpProtocol: pulumi.String("TCP"),
			PortRange:  pulumi.String("80"),
			Target:     httpProxy.SelfLink,
		})
		if err != nil {
			return err
		}
		ctx.Export("originURL", bucket.Name.ApplyT(func(name string) (string, error) {
			return fmt.Sprintf("https://storage.googleapis.com/%v/index.html", name), nil
		}).(pulumi.StringOutput))
		ctx.Export("cdnURL", ip.Address.ApplyT(func(address string) (string, error) {
			return fmt.Sprintf("http://%v", address), nil
		}).(pulumi.StringOutput))
		return nil
	})
}