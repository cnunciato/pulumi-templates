using System.Collections.Generic;
using Pulumi;
using Aws = Pulumi.Aws;
using SyncedFolder = Pulumi.SyncedFolder;

return await Deployment.RunAsync(() =>
{
    // Import the program's configuration settings.
    var config = new Config();
    var path = config.Get("path") ?? "./www";
    var indexDocument = config.Get("indexDocument") ?? "index.html";
    var errorDocument = config.Get("errorDocument") ?? "error.html";
    var domain = config.Require("domain");
    var subdomain = config.Require("subdomain");
    var domainName = $"{subdomain}.{domain}";

    // Create an S3 bucket and configure it as a website.
    var bucket = new Aws.S3.Bucket("bucket", new()
    {
        Acl = "public-read",
        Website = new Aws.S3.Inputs.BucketWebsiteArgs
        {
            IndexDocument = indexDocument,
            ErrorDocument = errorDocument,
        },
    });

    // Use a synced folder to manage the files of the website.
    var bucketFolder = new SyncedFolder.S3BucketFolder("bucket-folder", new()
    {
        Path = path,
        BucketName = bucket.BucketName,
        Acl = "public-read",
    });

    // Look up your existing Route 53-managed zone.
    var zone = Aws.Route53.GetZone.Invoke(new()
    {
        Name = domain,
    });

    // Provision a new ACM certificate.
    var certificate = new Aws.Acm.Certificate("certificate", new()
    {
        DomainName = domainName,
        ValidationMethod = "DNS",
    },
    // ACM certificates must be created in the us-east-1 region.
    new CustomResourceOptions {
        Provider = new Aws.Provider("us-east-provider", new() {
            Region = "us-east-1",
        })
    });

    // Validate the ACM certificate with DNS.
    var validationOption = certificate.DomainValidationOptions.GetAt(0);
    var certificateValidation = new Aws.Route53.Record("certificate-validation", new()
    {
        Name = validationOption.Apply(option => option.ResourceRecordName!),
        Type = validationOption.Apply(option => option.ResourceRecordType!),
        Records = new[]
        {
            validationOption.Apply(option => option.ResourceRecordValue!),
        },
        ZoneId = zone.Apply(zone => zone.ZoneId),
        Ttl = 60,
    });

    // Create a CloudFront CDN to distribute and cache the website.
    var cdn = new Aws.CloudFront.Distribution("cdn", new()
    {
        Enabled = true,
        Origins = new[]
        {
            new Aws.CloudFront.Inputs.DistributionOriginArgs
            {
                OriginId = bucket.Arn,
                DomainName = bucket.WebsiteEndpoint,
                CustomOriginConfig = new Aws.CloudFront.Inputs.DistributionOriginCustomOriginConfigArgs
                {
                    OriginProtocolPolicy = "http-only",
                    HttpPort = 80,
                    HttpsPort = 443,
                    OriginSslProtocols = new[]
                    {
                        "TLSv1.2",
                    },
                },
            },
        },
        DefaultCacheBehavior = new Aws.CloudFront.Inputs.DistributionDefaultCacheBehaviorArgs
        {
            TargetOriginId = bucket.Arn,
            ViewerProtocolPolicy = "redirect-to-https",
            AllowedMethods = new[]
            {
                "GET",
                "HEAD",
                "OPTIONS",
            },
            CachedMethods = new[]
            {
                "GET",
                "HEAD",
                "OPTIONS",
            },
            DefaultTtl = 600,
            MaxTtl = 600,
            MinTtl = 600,
            ForwardedValues = new Aws.CloudFront.Inputs.DistributionDefaultCacheBehaviorForwardedValuesArgs
            {
                QueryString = true,
                Cookies = new Aws.CloudFront.Inputs.DistributionDefaultCacheBehaviorForwardedValuesCookiesArgs
                {
                    Forward = "all",
                },
            },
        },
        PriceClass = "PriceClass_100",
        CustomErrorResponses = new[]
        {
            new Aws.CloudFront.Inputs.DistributionCustomErrorResponseArgs
            {
                ErrorCode = 404,
                ResponseCode = 404,
                ResponsePagePath = $"/{errorDocument}",
            },
        },
        Restrictions = new Aws.CloudFront.Inputs.DistributionRestrictionsArgs
        {
            GeoRestriction = new Aws.CloudFront.Inputs.DistributionRestrictionsGeoRestrictionArgs
            {
                RestrictionType = "none",
            },
        },
        Aliases = new[]
        {
            domainName
        },
        ViewerCertificate = new Aws.CloudFront.Inputs.DistributionViewerCertificateArgs
        {
            CloudfrontDefaultCertificate = false,
            AcmCertificateArn = certificate.Arn,
            SslSupportMethod = "sni-only",
        },
    });

    // Create a DNS A record to point to the CDN.
    var record = new Aws.Route53.Record(domainName, new()
    {
        Name = subdomain,
        ZoneId = zone.Apply(zone => zone.ZoneId),
        Type = "A",
        Aliases = new[]
        {
            new Aws.Route53.Inputs.RecordAliasArgs
            {
                Name = cdn.DomainName,
                ZoneId = cdn.HostedZoneId,
                EvaluateTargetHealth = true,
            }
        },
    },
    new CustomResourceOptions {
        DependsOn = certificate,
    });

    // Export the URLs and hostnames of the bucket and distribution.
    return new Dictionary<string, object?>
    {
        ["originURL"] = Output.Format($"http://{bucket.WebsiteEndpoint}"),
        ["originHostname"] = bucket.WebsiteEndpoint,
        ["cdnURL"] = Output.Format($"https://{cdn.DomainName}"),
        ["cdnHostname"] = cdn.DomainName,
        ["domainName"] = $"https://{subdomain}.{domain}",
    };
});
