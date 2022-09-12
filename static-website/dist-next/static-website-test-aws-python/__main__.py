import pulumi
import pulumi_aws as aws
import pulumi_synced_folder as synced_folder

# Import the program's configuration settings.
config = pulumi.Config()
path = config.get("path") or "./www"
index_document = config.get("indexDocument") or "index.html"
error_document = config.get("errorDocument") or "error.html"
domain = config.require("domain");
subdomain = config.require("subdomain");
domain_name = f"{subdomain}.{domain}";

# Create an S3 bucket and configure it as a website.
bucket = aws.s3.Bucket(
    "bucket",
    acl="public-read",
    website=aws.s3.BucketWebsiteArgs(
        index_document=index_document,
        error_document=error_document,
    ),
)

# Use a synced folder to manage the files of the website.
bucket_folder = synced_folder.S3BucketFolder(
    "bucket-folder", path=path, bucket_name=bucket.bucket, acl="public-read"
)

# Look up your existing Route 53 hosted zone.
zone = aws.route53.get_zone_output(name=domain)

# Provision a new ACM certificate.
certificate = aws.acm.Certificate(
    "certificate",
    domain_name=domain_name,
    validation_method="DNS",
    opts=pulumi.ResourceOptions(
        # ACM certificates must be created in the us-east-1 region.
        provider=aws.Provider("us-east-provider", region="us-east-1"),
    ),
);

# Validate the ACM certificate with DNS.
options = certificate.domain_validation_options.apply(lambda options: options[0])
certificate_validation = aws.route53.Record(
    "certificate-validation",
    name=options.resource_record_name,
    type=options.resource_record_type,
    records=[options.resource_record_value],
    zone_id=zone.zone_id,
    ttl=60,
);

# Create a CloudFront CDN to distribute and cache the website.
cdn = aws.cloudfront.Distribution(
    "cdn",
    enabled=True,
    origins=[
        aws.cloudfront.DistributionOriginArgs(
            origin_id=bucket.arn,
            domain_name=bucket.website_endpoint,
            custom_origin_config=aws.cloudfront.DistributionOriginCustomOriginConfigArgs(
                origin_protocol_policy="http-only",
                http_port=80,
                https_port=443,
                origin_ssl_protocols=["TLSv1.2"],
            ),
        )
    ],
    default_cache_behavior=aws.cloudfront.DistributionDefaultCacheBehaviorArgs(
        target_origin_id=bucket.arn,
        viewer_protocol_policy="redirect-to-https",
        allowed_methods=[
            "GET",
            "HEAD",
            "OPTIONS",
        ],
        cached_methods=[
            "GET",
            "HEAD",
            "OPTIONS",
        ],
        default_ttl=600,
        max_ttl=600,
        min_ttl=600,
        forwarded_values=aws.cloudfront.DistributionDefaultCacheBehaviorForwardedValuesArgs(
            query_string=True,
            cookies=aws.cloudfront.DistributionDefaultCacheBehaviorForwardedValuesCookiesArgs(
                forward="all",
            ),
        ),
    ),
    price_class="PriceClass_100",
    custom_error_responses=[
        aws.cloudfront.DistributionCustomErrorResponseArgs(
            error_code=404,
            response_code=404,
            response_page_path=f"/{error_document}",
        )
    ],
    restrictions=aws.cloudfront.DistributionRestrictionsArgs(
        geo_restriction=aws.cloudfront.DistributionRestrictionsGeoRestrictionArgs(
            restriction_type="none",
        ),
    ),
    aliases=[
        domain_name,
    ],
    viewer_certificate=aws.cloudfront.DistributionViewerCertificateArgs(
        cloudfront_default_certificate=False,
        acm_certificate_arn=certificate.arn,
        ssl_support_method="sni-only",
    ),
)

# Create a DNS A record to point to the CDN.
my_site = aws.route53.Record(domain_name,
    zone_id=zone.zone_id,
    name=subdomain,
    type="A",
    aliases=[
        aws.route53.RecordAliasArgs(
            name=cdn.domain_name,
            zone_id=cdn.hosted_zone_id,
            evaluate_target_health=True,
        )
    ],
    opts=pulumi.ResourceOptions(
        depends_on=certificate,
    ),
)

# Export the URLs and hostnames of the bucket and distribution.
pulumi.export("originURL", pulumi.Output.concat("http://", bucket.website_endpoint))
pulumi.export("originHostname", bucket.website_endpoint)
pulumi.export("cdnURL", pulumi.Output.concat("https://", cdn.domain_name))
pulumi.export("cdnHostname", cdn.domain_name)
pulumi.export("domainURL", "https://" + domain_name)
