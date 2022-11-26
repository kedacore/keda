package openstack

import (
	"fmt"

	"github.com/Huawei/gophercloud"
	"github.com/Huawei/gophercloud/auth/aksk"
	tokenAuth "github.com/Huawei/gophercloud/auth/token"
	tokens2 "github.com/Huawei/gophercloud/openstack/identity/v2/tokens"
	tokens3 "github.com/Huawei/gophercloud/openstack/identity/v3/tokens"

	"errors"
	"strings"
	"os"
)

// service have same endpoint address in different location
var endpointSchemaList = map[string]string{
	"COMPUTE":  "https://ecs.%(region)s.%(domain)s/v2/%(projectID)s/",
	"ECSV1.1":  "https://ecs.%(region)s.%(domain)s/v1.1/%(projectID)s/",
	"ECSV2":    "https://ecs.%(region)s.%(domain)s/v2/%(projectID)s/",
	"ECS":  "https://ecs.%(region)s.%(domain)s/v1/%(projectID)s/",
	"IMAGE":    "https://ims.%(region)s.%(domain)s/",
	"NETWORK":  "https://vpc.%(region)s.%(domain)s/",
	"VOLUMEV2": "https://evs.%(region)s.%(domain)s/v2/%(projectID)s/",
	//"ANTIDDOS":      "https://antiddos.%(region)s.%(domain)s/",
	//"BSS":           "https://bss.%(region)s.%(domain)s/",
	"BSSV1":     "https://bss.%(domain)s/v1.0/",
	"BSSINTLV1":  "https://bss.%(domain)s/v1.0/",
	"VPC":     "https://vpc.%(region)s.%(domain)s/v1/%(projectID)s/",
	"CESV1":   "https://ces.%(region)s.%(domain)s/V1.0/%(projectID)s/",
	"VPCV2.0": "https://vpc.%(region)s.%(domain)s/v2.0/%(projectID)s/",
	"ASV1":    "https://as.%(region)s.%(domain)s/autoscaling-api/v1/%(projectID)s/",
	"ASV2":    "https://as.%(region)s.%(domain)s/autoscaling-api/v2/%(projectID)s/",
	"DNS":     "https://dns.%(region)s.%(domain)s/",
	"FGSV2":   "https://functiongraph.%(region)s.%(domain)s/v2/%(projectID)s/",
	"RDSV3":   "https://rds.%(region)s.%(domain)s/v3/%(projectID)s/",
	"IDENTITY": "https://iam.%(domain)s/v3",
}

/*
V2EndpointURL discovers the endpoint URL for a specific service from a
ServiceCatalog acquired during the v2 identity service.

The specified EndpointOpts are used to identify a unique, unambiguous endpoint
to return. It's an error both when multiple endpoints match the provided
criteria and when none do. The minimum that can be specified is a Type, but you
will also often need to specify a Name and/or a Region depending on what's
available on your OpenStack deployment.
*/
func V2EndpointURL(catalog *tokens2.ServiceCatalog, opts gophercloud.EndpointOpts) (string, error) {
	// Extract Endpoints from the catalog entries that match the requested Type, Name if provided, and Region if provided.
	var endpoints = make([]tokens2.Endpoint, 0, 1)
	for _, entry := range catalog.Entries {
		if (entry.Type == opts.Type) && (opts.Name == "" || entry.Name == opts.Name) {
			for _, endpoint := range entry.Endpoints {
				if opts.Region == "" || endpoint.Region == opts.Region {
					endpoints = append(endpoints, endpoint)
				}
			}
		}
	}

	// Report an error if the options were ambiguous.
	if len(endpoints) > 1 {
		//		err := &ErrMultipleMatchingEndpointsV2{}
		//		err.Endpoints = endpoints
		//		return "", err

		message := fmt.Sprintf("Discovered %d matching endpoints: %#v", len(endpoints), endpoints)
		err := gophercloud.NewSystemCommonError("Com.2000", message)
		return "", err
	}

	// Extract the appropriate URL from the matching Endpoint.
	for _, endpoint := range endpoints {
		switch opts.Availability {
		case gophercloud.AvailabilityPublic:
			return gophercloud.NormalizeURL(endpoint.PublicURL), nil
		case gophercloud.AvailabilityInternal:
			return gophercloud.NormalizeURL(endpoint.InternalURL), nil
		case gophercloud.AvailabilityAdmin:
			return gophercloud.NormalizeURL(endpoint.AdminURL), nil
		default:
			//			err := &ErrInvalidAvailabilityProvided{}
			//			err.Argument = "Availability"
			//			err.Value = opts.Availability
			//			return "", err

			value := fmt.Sprintf("Availability:%+v", opts.Availability)
			message := fmt.Sprintf(gophercloud.CE_InvalidInputMessage, value)
			err := gophercloud.NewSystemCommonError(gophercloud.CE_InvalidInputCode, message)
			return "", err
		}
	}

	// Report an error if there were no matching endpoints.
	//	err := &gophercloud.ErrEndpointNotFound{}
	//	return "", err

	err := gophercloud.NewSystemCommonError(gophercloud.CE_NoEndPointInCatalogCode, gophercloud.CE_NoEndPointInCatalogMessage)
	return "", err
}

// Extract Endpoints from the catalog entries that match the requested Type, Interface, Name if provided, and Region if provided.
func V3ExtractEndpointURL(catalog *tokens3.ServiceCatalog, opts gophercloud.EndpointOpts, tokenOptions tokens3.AuthOptionsBuilder) (string, error) {

	if opts.Type == "" {
		return "", errors.New("Service type can not be empty.")
	}

	ss := strings.Replace(opts.Type, "-", "_", -1)
	key := fmt.Sprintf("SDK_%s_ENDPOINT_OVERRIDE", strings.ToUpper(ss))
	endpointFromEnv := os.Getenv(key)
	if endpointFromEnv != "" {
		if opts, ok := tokenOptions.(*tokenAuth.TokenOptions); ok {
			endpointFromEnv = strings.Replace(endpointFromEnv, "%(projectID)s", opts.ProjectID, 1)
			return endpointFromEnv, nil
		}
	}

	return V3EndpointURL(catalog, opts)
}

// Extract Endpoints from the catalog entries that match the requested Type, Interface, Name if provided, and Region if provided.
func V3TokenIdExtractEndpointURL(catalog *tokens3.ServiceCatalog, opts gophercloud.EndpointOpts, tokenIdOptions tokenAuth.TokenIdOptions) (string, error) {

	if opts.Type == "" {
		return "", errors.New("Service type can not be empty.")
	}

	ss := strings.Replace(opts.Type, "-", "_", -1)
	key := fmt.Sprintf("SDK_%s_ENDPOINT_OVERRIDE", strings.ToUpper(ss))
	endpointFromEnv := os.Getenv(key)
	if endpointFromEnv != "" {
		endpointFromEnv = strings.Replace(endpointFromEnv, "%(projectID)s", tokenIdOptions.ProjectID, 1)
		return endpointFromEnv, nil
	}

	return V3EndpointURL(catalog, opts)
}

/*
V3EndpointURL discovers the endpoint URL for a specific service from a Catalog
acquired during the v3 identity service.

The specified EndpointOpts are used to identify a unique, unambiguous endpoint
to return. It's an error both when multiple endpoints match the provided
criteria and when none do. The minimum that can be specified is a Type, but you
will also often need to specify a Name and/or a Region depending on what's
available on your OpenStack deployment.
*/
func V3EndpointURL(catalog *tokens3.ServiceCatalog, opts gophercloud.EndpointOpts) (string, error) {
	var endpoints = make([]tokens3.Endpoint, 0, 1)
	for _, entry := range catalog.Entries {
		if (entry.Type == opts.Type) && (opts.Name == "" || entry.Name == opts.Name) {
			for _, endpoint := range entry.Endpoints {
				if opts.Availability != gophercloud.AvailabilityAdmin &&
					opts.Availability != gophercloud.AvailabilityPublic &&
					opts.Availability != gophercloud.AvailabilityInternal {
					//					err := &ErrInvalidAvailabilityProvided{}
					//					err.Argument = "Availability"
					//					err.Value = opts.Availability
					//					return "", err

					value := fmt.Sprintf("Availability:%+v", opts.Availability)
					message := fmt.Sprintf(gophercloud.CE_InvalidInputMessage, value)
					err := gophercloud.NewSystemCommonError(gophercloud.CE_InvalidInputCode, message)
					return "", err
				}
				if (opts.Availability == gophercloud.Availability(endpoint.Interface)) &&
					(opts.Region == "" || endpoint.Region == opts.Region) {
					endpoints = append(endpoints, endpoint)
				}
			}
		}
	}

	// Report an error if the options were ambiguous.
	if len(endpoints) > 1 {
		//return "", ErrMultipleMatchingEndpointsV3{Endpoints: endpoints}

		message := fmt.Sprintf("Discovered %d matching endpoints: %#v", len(endpoints), endpoints)
		err := gophercloud.NewSystemCommonError("Com.2000", message)
		return "", err
	}

	// Extract the URL from the matching Endpoint.
	for _, endpoint := range endpoints {
		return gophercloud.NormalizeURL(endpoint.URL), nil
	}

	// Report an error if there were no matching endpoints.
	//	err := &gophercloud.ErrEndpointNotFound{}
	//	return "", err

	err := gophercloud.NewSystemCommonError(gophercloud.CE_NoEndPointInCatalogCode, gophercloud.CE_NoEndPointInCatalogMessage)
	return "", err
}

/*
   GetEndpointURLForAKSKAuth discovers the endpoint  from V3EndpointURL function firstly,
   if the endpoint is null then concat the service type and domain as the endpoint
*/
func GetEndpointURLForAKSKAuth(catalog *tokens3.ServiceCatalog, opts gophercloud.EndpointOpts, akskOptions aksk.AKSKOptions) (string, error) {

	if akskOptions.Cloud != "" {
		akskOptions.Domain = akskOptions.Cloud
	}

	if opts.Type == "" {
		return "", errors.New("Service type can not be empty.")
	}

	ss := strings.Replace(opts.Type, "-", "_", -1)
	key := fmt.Sprintf("SDK_%s_ENDPOINT_OVERRIDE", strings.ToUpper(ss))
	endpointFromEnv := os.Getenv(key)
	if endpointFromEnv != "" {
		//endpointFromEnv = strings.Replace(endpointFromEnv, "%(region)s", akskOptions.Region, 1)
		//endpointFromEnv = strings.Replace(endpointFromEnv, "%(domain)s", akskOptions.Domain, 1)
		endpointFromEnv = strings.Replace(endpointFromEnv, "%(projectID)s", akskOptions.ProjectID, 1)
		return endpointFromEnv, nil
	}

	endpoint, err := V3EndpointURL(catalog, opts)
	if err != nil || endpoint == "" {
		if akskOptions.Domain == "" {
			return "", errors.New("AKSKOptions.Cloud can not be empty.")
		}

		region := opts.Region

		if region == "" {
			region = akskOptions.Region
		}

		if endpointSchema, ok := endpointSchemaList[strings.ToUpper(opts.Type)]; ok {
			endpoint = strings.Replace(endpointSchema, "%(domain)s", akskOptions.Domain, 1)
			endpoint = strings.Replace(endpoint, "%(region)s", region, 1)
			endpoint = strings.Replace(endpoint, "%(projectID)s", akskOptions.ProjectID, 1)

			return endpoint, nil
		}
	}

	return endpoint, err
}

/*
   GetEndpointURLForAKSKAuth discovers the endpoint  from V3EndpointURL function firstly,
   if the endpoint is null then concat the service type and domain as the endpoint
*/
/*
func GetEndpointURLForAKSKAuth(catalog *tokens3.ServiceCatalog, opts gophercloud.EndpointOpts, akskOptions aksk.AKSKOptions) (string, error) {
	if opts.Region == "" {
		opts.Region = akskOptions.Region
	}

	endpoint, err := V3EndpointURL(catalog, opts)
	//fmt.Println("EDP0", endpoint)
	if err != nil || endpoint == "" {
		if akskOptions.Domain == "" {
			return "", errors.New("ServiceDomainName can not be empty.")
		}
		//fmt.Println("EDP1", endpoint)
		if endpointSchema, ok := endpointSchemaList[strings.ToUpper(opts.Type)]; ok {
			endpoint = strings.Replace(endpointSchema, "%(domain)s", akskOptions.Domain, 1)
			endpoint = strings.Replace(endpoint, "%(region)s", opts.Region, 1)
			endpoint = strings.Replace(endpoint, "%(projectID)s", akskOptions.ProjectID, 1)
			//fmt.Println("EDP2", endpoint)
			return endpoint, nil
		}
	}

	return endpoint, err
}
*/