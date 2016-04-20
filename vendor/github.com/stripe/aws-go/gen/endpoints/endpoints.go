// Package endpoints provides lookups for all AWS service endpoints.
package endpoints

import (
	"strings"
)

// Lookup returns the endpoint for the given service in the given region plus
// any overrides for the service name and region.
func Lookup(service, region string) (uri, newService, newRegion string) {
	switch service {

	case "cloudfront":

		if !strings.HasPrefix(region, "cn-") {
			return format("https://cloudfront.amazonaws.com", service, region), service, "us-east-1"
		}

	case "dynamodb":

		if region == "local" {
			return format("http://localhost:8000", service, region), "dynamodb", "us-east-1"
		}

	case "elasticmapreduce":

		if strings.HasPrefix(region, "cn-") {
			return format("https://elasticmapreduce.cn-north-1.amazonaws.com.cn", service, region), service, region
		}

		if region == "eu-central-1" {
			return format("https://elasticmapreduce.eu-central-1.amazonaws.com", service, region), service, region
		}

		if region == "us-east-1" {
			return format("https://elasticmapreduce.us-east-1.amazonaws.com", service, region), service, region
		}

		if region != "" {
			return format("https://{region}.elasticmapreduce.amazonaws.com", service, region), service, region
		}

	case "iam":

		if strings.HasPrefix(region, "cn-") {
			return format("https://{service}.cn-north-1.amazonaws.com.cn", service, region), service, region
		}

		if strings.HasPrefix(region, "us-gov") {
			return format("https://{service}.us-gov.amazonaws.com", service, region), service, region
		}

		return format("https://iam.amazonaws.com", service, region), service, "us-east-1"

	case "importexport":

		if !strings.HasPrefix(region, "cn-") {
			return format("https://importexport.amazonaws.com", service, region), service, region
		}

	case "rds":

		if region == "us-east-1" {
			return format("https://rds.amazonaws.com", service, region), service, region
		}

	case "route53":

		if !strings.HasPrefix(region, "cn-") {
			return format("https://route53.amazonaws.com", service, region), service, region
		}

	case "s3":

		if region == "us-east-1" || region == "" {
			return format("{scheme}://s3.amazonaws.com", service, region), service, "us-east-1"
		}

		if strings.HasPrefix(region, "cn-") {
			return format("{scheme}://{service}.{region}.amazonaws.com.cn", service, region), service, region
		}

		if region == "us-east-1" || region == "ap-northeast-1" || region == "sa-east-1" || region == "ap-southeast-1" || region == "ap-southeast-2" || region == "us-west-2" || region == "us-west-1" || region == "eu-west-1" || region == "us-gov-west-1" || region == "fips-us-gov-west-1" {
			return format("{scheme}://{service}-{region}.amazonaws.com", service, region), service, region
		}

		if region != "" {
			return format("{scheme}://{service}.{region}.amazonaws.com", service, region), service, region
		}

	case "sdb":

		if region == "us-east-1" {
			return format("https://sdb.amazonaws.com", service, region), service, region
		}

	case "sqs":

		if region == "us-east-1" {
			return format("https://queue.amazonaws.com", service, region), service, region
		}

		if strings.HasPrefix(region, "cn-") {
			return format("https://{region}.queue.amazonaws.com.cn", service, region), service, region
		}

		if region != "" {
			return format("https://{region}.queue.amazonaws.com", service, region), service, region
		}

	case "sts":

		if strings.HasPrefix(region, "cn-") {
			return format("{scheme}://{service}.cn-north-1.amazonaws.com.cn", service, region), service, region
		}

		if strings.HasPrefix(region, "us-gov") {
			return format("https://{service}.{region}.amazonaws.com", service, region), service, region
		}

		return format("https://sts.amazonaws.com", service, region), service, "us-east-1"

	}

	if strings.HasPrefix(region, "cn-") {
		return format("{scheme}://{service}.{region}.amazonaws.com.cn", service, region), service, region
	}

	if region != "" {
		return format("{scheme}://{service}.{region}.amazonaws.com", service, region), service, region
	}

	panic("unknown endpoint for " + service + " in " + region)
}

func format(uri, service, region string) string {
	uri = strings.Replace(uri, "{scheme}", "https", -1)
	uri = strings.Replace(uri, "{service}", service, -1)
	uri = strings.Replace(uri, "{region}", region, -1)
	return uri
}
