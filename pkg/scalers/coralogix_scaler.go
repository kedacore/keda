package scalers

import (
	"fmt"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"reflect"
)

const (
	scalerTypeKey           = "scalerType"
	promScaler              = "prometheus_scaler"
	domainNameKey           = "domain"
	tokenKey                = "token"
	tokenHeaderStringFmtKey = "token_header_fmt"
	promServerAddressFmt    = "prometheusAPIStringFmt"

	tokenHeaderStringFmtStringFmt = "token=%s"
	promAPIStringFmt              = "https://prom-api.%s"
)

type CoralogixMetadata struct {
	domain            string
	token             string
	scalerType        string
	promServerAddress string
}

func NewCoralogixScaler(config *ScalerConfig) (Scaler, error) {
	coralogixMetadata, err := parseCoralogixMetadata(config)
	if err != nil {
		return nil, err
	}

	scalerMap := map[string]func(*ScalerConfig, *CoralogixMetadata) (Scaler, error){
		promScaler: NewPromScaler,
	}

	if v, ok := scalerMap[coralogixMetadata.scalerType]; ok {
		return v(config, coralogixMetadata)
	}

	return nil, fmt.Errorf("%s must be one of the following option %s, the provided value is %s", scalerTypeKey, reflect.ValueOf(scalerMap).MapKeys(), coralogixMetadata.scalerType)
}

func OverloadPrometheusMetadata(config *ScalerConfig, coralogixMetadata *CoralogixMetadata) *ScalerConfig {
	parsedCustomHeaders, _ := kedautil.ParseStringList(config.TriggerMetadata[promCustomHeaders])
	if _, ok := parsedCustomHeaders[tokenKey]; !ok {
		if val, ok := config.TriggerMetadata[promCustomHeaders]; ok && val != "" {
			config.TriggerMetadata[promCustomHeaders] = fmt.Sprintf("%s,%s", val, coralogixMetadata.token)
		} else {
			config.TriggerMetadata[promCustomHeaders] = coralogixMetadata.token
		}
	}

	config.TriggerMetadata[promServerAddress] = coralogixMetadata.promServerAddress
	return config
}

func NewPromScaler(config *ScalerConfig, coralogixMetadata *CoralogixMetadata) (Scaler, error) {
	return NewPrometheusScaler(OverloadPrometheusMetadata(config, coralogixMetadata))
}

func parseCoralogixMetadata(config *ScalerConfig) (meta *CoralogixMetadata, err error) {
	domain, err := GetFromAuthOrMeta(config, domainNameKey)
	if err != nil {
		return nil, err
	}
	token, err := GetFromAuthOrMeta(config, tokenKey)
	if err != nil {
		return nil, err
	}
	scalerType, err := GetFromAuthOrMeta(config, scalerTypeKey)
	if err != nil {
		return nil, err
	}

	tokenHeaderStringFmt, ok := config.TriggerMetadata[tokenHeaderStringFmtKey]
	if !ok {
		tokenHeaderStringFmt = tokenHeaderStringFmtStringFmt
	}
	tokenHeaderParsed := fmt.Sprintf(tokenHeaderStringFmt, token)

	prometheusAPIStringFmt, ok := config.TriggerMetadata[promServerAddressFmt]
	if !ok {
		prometheusAPIStringFmt = promAPIStringFmt
	}
	prometheusServers := fmt.Sprintf(prometheusAPIStringFmt, domain)

	return &CoralogixMetadata{
		domain:            domain,
		token:             tokenHeaderParsed,
		scalerType:        scalerType,
		promServerAddress: prometheusServers,
	}, nil
}
