import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

const config = new pulumi.Config();
const path = config.get("path") || "./www";
const indexDocument = config.get("indexDocument") || "index.html";
const errorDocument = config.get("errorDocument") || "error.html";
const domain = config.require("domain");
const subdomain = config.require("subdomain");
const domainName = `${subdomain}.${domain}`;
const zone = aws.route53.getZone({
    name: domain,
});
const usEastProvider = new aws.Provider("us-east-provider", {region: "us-east-1"});
const certificate = new aws.acm.Certificate("certificate", {
    domainName: domainName,
    validationMethod: "DNS",
}, {
    provider: usEastProvider,
});
const validationOptions = certificate.domainValidationOptions;
const certificateValidation = new aws.route53.Record("certificateValidation", {
    name: validationOptions.apply(validationOptions => validationOptions[0].resourceRecordName),
    type: validationOptions.apply(validationOptions => validationOptions[0].resourceRecordType).apply((x) => aws.route53.recordtype.RecordType[x]),
    zoneId: zone.then(zone => zone.zoneId),
    ttl: 60,
    records: [validationOptions.apply(validationOptions => validationOptions[0].resourceRecordValue)],
});
