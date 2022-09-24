import pulumi
import pulumi_azure_native as azure_native
import pulumi_synced_folder as synced_folder

config = pulumi.Config()
path = config.get("path") or "./www"
index_document = config.get("indexDocument") or "index.html"
error_document = config.get("errorDocument") or "error.html"

resource_group = azure_native.resources.ResourceGroup("resource-group")

account = azure_native.storage.StorageAccount("account",
    resource_group_name=resource_group.name,
    kind="StorageV2",
    sku=azure_native.storage.SkuArgs(
        name="Standard_LRS",
    ))

container = azure_native.storage.BlobContainer("container",
    account_name=account.name,
    resource_group_name=resource_group.name,
    public_access=azure_native.storage.PublicAccess.NONE)

blob_sas = pulumi.Output.all(resource_group.name, account.name, container.name).apply(lambda args: azure_native.storage.list_storage_account_service_sas_output(
        resource_group_name=args[0],
        account_name=args[1],
        protocols=azure_native.storage.HttpProtocol.HTTPS,
        shared_access_start_time="2022-01-01",
        shared_access_expiry_time="2030-01-01",
        resource="c",
        permissions="r",
        canonicalized_resource=f"/blob/{args[1]}/{args[2]}",
        content_type="application/json",
        cache_control="max-age=5",
        content_disposition="inline",
        content_encoding="deflate"))

website = azure_native.storage.StorageAccountStaticWebsite("website",
    account_name=account.name,
    resource_group_name=resource_group.name,
    index_document=index_document,
    error404_document=error_document)

synced_folder = synced_folder.AzureBlobFolder("synced-folder",
    path=path,
    resource_group_name=resource_group.name,
    storage_account_name=account.name,
    container_name=website.container_name)

plan = azure_native.web.AppServicePlan("plan",
    resource_group_name=resource_group.name,
    sku=azure_native.web.SkuDescriptionArgs(
        name="Y1",
        tier="Dynamic",
    ))

blob = azure_native.storage.Blob("blob",
    account_name=account.name,
    resource_group_name=resource_group.name,
    container_name=container.name,
    source=pulumi.FileArchive("./api"))

app = azure_native.web.WebApp("app",
    resource_group_name=resource_group.name,
    server_farm_id=plan.id,
    kind="FunctionApp",
    site_config=azure_native.web.SiteConfigArgs(
        app_settings=[
            azure_native.web.NameValuePairArgs(
                name="FUNCTIONS_WORKER_RUNTIME",
                value="node",
            ),
            azure_native.web.NameValuePairArgs(
                name="WEBSITE_NODE_DEFAULT_VERSION",
                value="~14",
            ),
            azure_native.web.NameValuePairArgs(
                name="FUNCTIONS_EXTENSION_VERSION",
                value="~3",
            ),
            azure_native.web.NameValuePairArgs(
                name="WEBSITE_RUN_FROM_PACKAGE",
                value=pulumi.Output.all(account.name, container.name, blob.name, blob_sas).apply(
                    lambda args: f"https://{args[0]}.blob.core.windows.net/{args[1]}/{args[2]}?{args[3].service_sas_token}"),
            ),
        ],
        cors=azure_native.web.CorsSettingsArgs(
            allowed_origins=["*"],
        ),
    ))

site_config = azure_native.storage.Blob("config.json",
    account_name=account.name,
    resource_group_name=resource_group.name,
    container_name=website.container_name,
    content_type="application/json",
    source=app.default_host_name.apply(lambda hostname: pulumi.StringAsset("{ \"api\": \"https://" + hostname  + "/api\" }")))

pulumi.export("originURL", account.primary_endpoints.web)
pulumi.export("apiURL", app.default_host_name.apply(lambda default_host_name: f"https://{default_host_name}/api"))
