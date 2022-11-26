// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsTestRequest Object describing the Synthetic test request.
type SyntheticsTestRequest struct {
	// Allows loading insecure content for an HTTP request in a multistep test step.
	AllowInsecure *bool `json:"allow_insecure,omitempty"`
	// Object to handle basic authentication when performing the test.
	BasicAuth *SyntheticsBasicAuth `json:"basicAuth,omitempty"`
	// Body to include in the test.
	Body *string `json:"body,omitempty"`
	// Client certificate to use when performing the test request.
	Certificate *SyntheticsTestRequestCertificate `json:"certificate,omitempty"`
	// DNS server to use for DNS tests.
	DnsServer *string `json:"dnsServer,omitempty"`
	// DNS server port to use for DNS tests.
	DnsServerPort *int32 `json:"dnsServerPort,omitempty"`
	// Specifies whether or not the request follows redirects.
	FollowRedirects *bool `json:"follow_redirects,omitempty"`
	// Headers to include when performing the test.
	Headers map[string]string `json:"headers,omitempty"`
	// Host name to perform the test with.
	Host *string `json:"host,omitempty"`
	// Message to send for UDP or WebSocket tests.
	Message *string `json:"message,omitempty"`
	// Metadata to include when performing the gRPC test.
	Metadata map[string]string `json:"metadata,omitempty"`
	// The HTTP method.
	Method *HTTPMethod `json:"method,omitempty"`
	// Determines whether or not to save the response body.
	NoSavingResponseBody *bool `json:"noSavingResponseBody,omitempty"`
	// Number of pings to use per test.
	NumberOfPackets *int32 `json:"numberOfPackets,omitempty"`
	// Port to use when performing the test.
	Port *int64 `json:"port,omitempty"`
	// The proxy to perform the test.
	Proxy *SyntheticsTestRequestProxy `json:"proxy,omitempty"`
	// Query to use for the test.
	Query interface{} `json:"query,omitempty"`
	// For SSL tests, it specifies on which server you want to initiate the TLS handshake,
	// allowing the server to present one of multiple possible certificates on
	// the same IP address and TCP port number.
	Servername *string `json:"servername,omitempty"`
	// gRPC service on which you want to perform the healthcheck.
	Service *string `json:"service,omitempty"`
	// Turns on a traceroute probe to discover all gateways along the path to the host destination.
	ShouldTrackHops *bool `json:"shouldTrackHops,omitempty"`
	// Timeout in seconds for the test.
	Timeout *float64 `json:"timeout,omitempty"`
	// URL to perform the test with.
	Url *string `json:"url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestRequest instantiates a new SyntheticsTestRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestRequest() *SyntheticsTestRequest {
	this := SyntheticsTestRequest{}
	return &this
}

// NewSyntheticsTestRequestWithDefaults instantiates a new SyntheticsTestRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestRequestWithDefaults() *SyntheticsTestRequest {
	this := SyntheticsTestRequest{}
	return &this
}

// GetAllowInsecure returns the AllowInsecure field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetAllowInsecure() bool {
	if o == nil || o.AllowInsecure == nil {
		var ret bool
		return ret
	}
	return *o.AllowInsecure
}

// GetAllowInsecureOk returns a tuple with the AllowInsecure field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetAllowInsecureOk() (*bool, bool) {
	if o == nil || o.AllowInsecure == nil {
		return nil, false
	}
	return o.AllowInsecure, true
}

// HasAllowInsecure returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasAllowInsecure() bool {
	if o != nil && o.AllowInsecure != nil {
		return true
	}

	return false
}

// SetAllowInsecure gets a reference to the given bool and assigns it to the AllowInsecure field.
func (o *SyntheticsTestRequest) SetAllowInsecure(v bool) {
	o.AllowInsecure = &v
}

// GetBasicAuth returns the BasicAuth field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetBasicAuth() SyntheticsBasicAuth {
	if o == nil || o.BasicAuth == nil {
		var ret SyntheticsBasicAuth
		return ret
	}
	return *o.BasicAuth
}

// GetBasicAuthOk returns a tuple with the BasicAuth field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetBasicAuthOk() (*SyntheticsBasicAuth, bool) {
	if o == nil || o.BasicAuth == nil {
		return nil, false
	}
	return o.BasicAuth, true
}

// HasBasicAuth returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasBasicAuth() bool {
	if o != nil && o.BasicAuth != nil {
		return true
	}

	return false
}

// SetBasicAuth gets a reference to the given SyntheticsBasicAuth and assigns it to the BasicAuth field.
func (o *SyntheticsTestRequest) SetBasicAuth(v SyntheticsBasicAuth) {
	o.BasicAuth = &v
}

// GetBody returns the Body field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetBody() string {
	if o == nil || o.Body == nil {
		var ret string
		return ret
	}
	return *o.Body
}

// GetBodyOk returns a tuple with the Body field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetBodyOk() (*string, bool) {
	if o == nil || o.Body == nil {
		return nil, false
	}
	return o.Body, true
}

// HasBody returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasBody() bool {
	if o != nil && o.Body != nil {
		return true
	}

	return false
}

// SetBody gets a reference to the given string and assigns it to the Body field.
func (o *SyntheticsTestRequest) SetBody(v string) {
	o.Body = &v
}

// GetCertificate returns the Certificate field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetCertificate() SyntheticsTestRequestCertificate {
	if o == nil || o.Certificate == nil {
		var ret SyntheticsTestRequestCertificate
		return ret
	}
	return *o.Certificate
}

// GetCertificateOk returns a tuple with the Certificate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetCertificateOk() (*SyntheticsTestRequestCertificate, bool) {
	if o == nil || o.Certificate == nil {
		return nil, false
	}
	return o.Certificate, true
}

// HasCertificate returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasCertificate() bool {
	if o != nil && o.Certificate != nil {
		return true
	}

	return false
}

// SetCertificate gets a reference to the given SyntheticsTestRequestCertificate and assigns it to the Certificate field.
func (o *SyntheticsTestRequest) SetCertificate(v SyntheticsTestRequestCertificate) {
	o.Certificate = &v
}

// GetDnsServer returns the DnsServer field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetDnsServer() string {
	if o == nil || o.DnsServer == nil {
		var ret string
		return ret
	}
	return *o.DnsServer
}

// GetDnsServerOk returns a tuple with the DnsServer field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetDnsServerOk() (*string, bool) {
	if o == nil || o.DnsServer == nil {
		return nil, false
	}
	return o.DnsServer, true
}

// HasDnsServer returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasDnsServer() bool {
	if o != nil && o.DnsServer != nil {
		return true
	}

	return false
}

// SetDnsServer gets a reference to the given string and assigns it to the DnsServer field.
func (o *SyntheticsTestRequest) SetDnsServer(v string) {
	o.DnsServer = &v
}

// GetDnsServerPort returns the DnsServerPort field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetDnsServerPort() int32 {
	if o == nil || o.DnsServerPort == nil {
		var ret int32
		return ret
	}
	return *o.DnsServerPort
}

// GetDnsServerPortOk returns a tuple with the DnsServerPort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetDnsServerPortOk() (*int32, bool) {
	if o == nil || o.DnsServerPort == nil {
		return nil, false
	}
	return o.DnsServerPort, true
}

// HasDnsServerPort returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasDnsServerPort() bool {
	if o != nil && o.DnsServerPort != nil {
		return true
	}

	return false
}

// SetDnsServerPort gets a reference to the given int32 and assigns it to the DnsServerPort field.
func (o *SyntheticsTestRequest) SetDnsServerPort(v int32) {
	o.DnsServerPort = &v
}

// GetFollowRedirects returns the FollowRedirects field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetFollowRedirects() bool {
	if o == nil || o.FollowRedirects == nil {
		var ret bool
		return ret
	}
	return *o.FollowRedirects
}

// GetFollowRedirectsOk returns a tuple with the FollowRedirects field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetFollowRedirectsOk() (*bool, bool) {
	if o == nil || o.FollowRedirects == nil {
		return nil, false
	}
	return o.FollowRedirects, true
}

// HasFollowRedirects returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasFollowRedirects() bool {
	if o != nil && o.FollowRedirects != nil {
		return true
	}

	return false
}

// SetFollowRedirects gets a reference to the given bool and assigns it to the FollowRedirects field.
func (o *SyntheticsTestRequest) SetFollowRedirects(v bool) {
	o.FollowRedirects = &v
}

// GetHeaders returns the Headers field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetHeaders() map[string]string {
	if o == nil || o.Headers == nil {
		var ret map[string]string
		return ret
	}
	return o.Headers
}

// GetHeadersOk returns a tuple with the Headers field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetHeadersOk() (*map[string]string, bool) {
	if o == nil || o.Headers == nil {
		return nil, false
	}
	return &o.Headers, true
}

// HasHeaders returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasHeaders() bool {
	if o != nil && o.Headers != nil {
		return true
	}

	return false
}

// SetHeaders gets a reference to the given map[string]string and assigns it to the Headers field.
func (o *SyntheticsTestRequest) SetHeaders(v map[string]string) {
	o.Headers = v
}

// GetHost returns the Host field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetHost() string {
	if o == nil || o.Host == nil {
		var ret string
		return ret
	}
	return *o.Host
}

// GetHostOk returns a tuple with the Host field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetHostOk() (*string, bool) {
	if o == nil || o.Host == nil {
		return nil, false
	}
	return o.Host, true
}

// HasHost returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasHost() bool {
	if o != nil && o.Host != nil {
		return true
	}

	return false
}

// SetHost gets a reference to the given string and assigns it to the Host field.
func (o *SyntheticsTestRequest) SetHost(v string) {
	o.Host = &v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *SyntheticsTestRequest) SetMessage(v string) {
	o.Message = &v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetMetadata() map[string]string {
	if o == nil || o.Metadata == nil {
		var ret map[string]string
		return ret
	}
	return o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetMetadataOk() (*map[string]string, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return &o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given map[string]string and assigns it to the Metadata field.
func (o *SyntheticsTestRequest) SetMetadata(v map[string]string) {
	o.Metadata = v
}

// GetMethod returns the Method field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetMethod() HTTPMethod {
	if o == nil || o.Method == nil {
		var ret HTTPMethod
		return ret
	}
	return *o.Method
}

// GetMethodOk returns a tuple with the Method field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetMethodOk() (*HTTPMethod, bool) {
	if o == nil || o.Method == nil {
		return nil, false
	}
	return o.Method, true
}

// HasMethod returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasMethod() bool {
	if o != nil && o.Method != nil {
		return true
	}

	return false
}

// SetMethod gets a reference to the given HTTPMethod and assigns it to the Method field.
func (o *SyntheticsTestRequest) SetMethod(v HTTPMethod) {
	o.Method = &v
}

// GetNoSavingResponseBody returns the NoSavingResponseBody field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetNoSavingResponseBody() bool {
	if o == nil || o.NoSavingResponseBody == nil {
		var ret bool
		return ret
	}
	return *o.NoSavingResponseBody
}

// GetNoSavingResponseBodyOk returns a tuple with the NoSavingResponseBody field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetNoSavingResponseBodyOk() (*bool, bool) {
	if o == nil || o.NoSavingResponseBody == nil {
		return nil, false
	}
	return o.NoSavingResponseBody, true
}

// HasNoSavingResponseBody returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasNoSavingResponseBody() bool {
	if o != nil && o.NoSavingResponseBody != nil {
		return true
	}

	return false
}

// SetNoSavingResponseBody gets a reference to the given bool and assigns it to the NoSavingResponseBody field.
func (o *SyntheticsTestRequest) SetNoSavingResponseBody(v bool) {
	o.NoSavingResponseBody = &v
}

// GetNumberOfPackets returns the NumberOfPackets field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetNumberOfPackets() int32 {
	if o == nil || o.NumberOfPackets == nil {
		var ret int32
		return ret
	}
	return *o.NumberOfPackets
}

// GetNumberOfPacketsOk returns a tuple with the NumberOfPackets field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetNumberOfPacketsOk() (*int32, bool) {
	if o == nil || o.NumberOfPackets == nil {
		return nil, false
	}
	return o.NumberOfPackets, true
}

// HasNumberOfPackets returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasNumberOfPackets() bool {
	if o != nil && o.NumberOfPackets != nil {
		return true
	}

	return false
}

// SetNumberOfPackets gets a reference to the given int32 and assigns it to the NumberOfPackets field.
func (o *SyntheticsTestRequest) SetNumberOfPackets(v int32) {
	o.NumberOfPackets = &v
}

// GetPort returns the Port field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetPort() int64 {
	if o == nil || o.Port == nil {
		var ret int64
		return ret
	}
	return *o.Port
}

// GetPortOk returns a tuple with the Port field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetPortOk() (*int64, bool) {
	if o == nil || o.Port == nil {
		return nil, false
	}
	return o.Port, true
}

// HasPort returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasPort() bool {
	if o != nil && o.Port != nil {
		return true
	}

	return false
}

// SetPort gets a reference to the given int64 and assigns it to the Port field.
func (o *SyntheticsTestRequest) SetPort(v int64) {
	o.Port = &v
}

// GetProxy returns the Proxy field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetProxy() SyntheticsTestRequestProxy {
	if o == nil || o.Proxy == nil {
		var ret SyntheticsTestRequestProxy
		return ret
	}
	return *o.Proxy
}

// GetProxyOk returns a tuple with the Proxy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetProxyOk() (*SyntheticsTestRequestProxy, bool) {
	if o == nil || o.Proxy == nil {
		return nil, false
	}
	return o.Proxy, true
}

// HasProxy returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasProxy() bool {
	if o != nil && o.Proxy != nil {
		return true
	}

	return false
}

// SetProxy gets a reference to the given SyntheticsTestRequestProxy and assigns it to the Proxy field.
func (o *SyntheticsTestRequest) SetProxy(v SyntheticsTestRequestProxy) {
	o.Proxy = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetQuery() interface{} {
	if o == nil || o.Query == nil {
		var ret interface{}
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetQueryOk() (*interface{}, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return &o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasQuery() bool {
	if o != nil && o.Query != nil {
		return true
	}

	return false
}

// SetQuery gets a reference to the given interface{} and assigns it to the Query field.
func (o *SyntheticsTestRequest) SetQuery(v interface{}) {
	o.Query = v
}

// GetServername returns the Servername field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetServername() string {
	if o == nil || o.Servername == nil {
		var ret string
		return ret
	}
	return *o.Servername
}

// GetServernameOk returns a tuple with the Servername field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetServernameOk() (*string, bool) {
	if o == nil || o.Servername == nil {
		return nil, false
	}
	return o.Servername, true
}

// HasServername returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasServername() bool {
	if o != nil && o.Servername != nil {
		return true
	}

	return false
}

// SetServername gets a reference to the given string and assigns it to the Servername field.
func (o *SyntheticsTestRequest) SetServername(v string) {
	o.Servername = &v
}

// GetService returns the Service field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetService() string {
	if o == nil || o.Service == nil {
		var ret string
		return ret
	}
	return *o.Service
}

// GetServiceOk returns a tuple with the Service field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetServiceOk() (*string, bool) {
	if o == nil || o.Service == nil {
		return nil, false
	}
	return o.Service, true
}

// HasService returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasService() bool {
	if o != nil && o.Service != nil {
		return true
	}

	return false
}

// SetService gets a reference to the given string and assigns it to the Service field.
func (o *SyntheticsTestRequest) SetService(v string) {
	o.Service = &v
}

// GetShouldTrackHops returns the ShouldTrackHops field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetShouldTrackHops() bool {
	if o == nil || o.ShouldTrackHops == nil {
		var ret bool
		return ret
	}
	return *o.ShouldTrackHops
}

// GetShouldTrackHopsOk returns a tuple with the ShouldTrackHops field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetShouldTrackHopsOk() (*bool, bool) {
	if o == nil || o.ShouldTrackHops == nil {
		return nil, false
	}
	return o.ShouldTrackHops, true
}

// HasShouldTrackHops returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasShouldTrackHops() bool {
	if o != nil && o.ShouldTrackHops != nil {
		return true
	}

	return false
}

// SetShouldTrackHops gets a reference to the given bool and assigns it to the ShouldTrackHops field.
func (o *SyntheticsTestRequest) SetShouldTrackHops(v bool) {
	o.ShouldTrackHops = &v
}

// GetTimeout returns the Timeout field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetTimeout() float64 {
	if o == nil || o.Timeout == nil {
		var ret float64
		return ret
	}
	return *o.Timeout
}

// GetTimeoutOk returns a tuple with the Timeout field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetTimeoutOk() (*float64, bool) {
	if o == nil || o.Timeout == nil {
		return nil, false
	}
	return o.Timeout, true
}

// HasTimeout returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasTimeout() bool {
	if o != nil && o.Timeout != nil {
		return true
	}

	return false
}

// SetTimeout gets a reference to the given float64 and assigns it to the Timeout field.
func (o *SyntheticsTestRequest) SetTimeout(v float64) {
	o.Timeout = &v
}

// GetUrl returns the Url field value if set, zero value otherwise.
func (o *SyntheticsTestRequest) GetUrl() string {
	if o == nil || o.Url == nil {
		var ret string
		return ret
	}
	return *o.Url
}

// GetUrlOk returns a tuple with the Url field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestRequest) GetUrlOk() (*string, bool) {
	if o == nil || o.Url == nil {
		return nil, false
	}
	return o.Url, true
}

// HasUrl returns a boolean if a field has been set.
func (o *SyntheticsTestRequest) HasUrl() bool {
	if o != nil && o.Url != nil {
		return true
	}

	return false
}

// SetUrl gets a reference to the given string and assigns it to the Url field.
func (o *SyntheticsTestRequest) SetUrl(v string) {
	o.Url = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AllowInsecure != nil {
		toSerialize["allow_insecure"] = o.AllowInsecure
	}
	if o.BasicAuth != nil {
		toSerialize["basicAuth"] = o.BasicAuth
	}
	if o.Body != nil {
		toSerialize["body"] = o.Body
	}
	if o.Certificate != nil {
		toSerialize["certificate"] = o.Certificate
	}
	if o.DnsServer != nil {
		toSerialize["dnsServer"] = o.DnsServer
	}
	if o.DnsServerPort != nil {
		toSerialize["dnsServerPort"] = o.DnsServerPort
	}
	if o.FollowRedirects != nil {
		toSerialize["follow_redirects"] = o.FollowRedirects
	}
	if o.Headers != nil {
		toSerialize["headers"] = o.Headers
	}
	if o.Host != nil {
		toSerialize["host"] = o.Host
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	if o.Method != nil {
		toSerialize["method"] = o.Method
	}
	if o.NoSavingResponseBody != nil {
		toSerialize["noSavingResponseBody"] = o.NoSavingResponseBody
	}
	if o.NumberOfPackets != nil {
		toSerialize["numberOfPackets"] = o.NumberOfPackets
	}
	if o.Port != nil {
		toSerialize["port"] = o.Port
	}
	if o.Proxy != nil {
		toSerialize["proxy"] = o.Proxy
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.Servername != nil {
		toSerialize["servername"] = o.Servername
	}
	if o.Service != nil {
		toSerialize["service"] = o.Service
	}
	if o.ShouldTrackHops != nil {
		toSerialize["shouldTrackHops"] = o.ShouldTrackHops
	}
	if o.Timeout != nil {
		toSerialize["timeout"] = o.Timeout
	}
	if o.Url != nil {
		toSerialize["url"] = o.Url
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AllowInsecure        *bool                             `json:"allow_insecure,omitempty"`
		BasicAuth            *SyntheticsBasicAuth              `json:"basicAuth,omitempty"`
		Body                 *string                           `json:"body,omitempty"`
		Certificate          *SyntheticsTestRequestCertificate `json:"certificate,omitempty"`
		DnsServer            *string                           `json:"dnsServer,omitempty"`
		DnsServerPort        *int32                            `json:"dnsServerPort,omitempty"`
		FollowRedirects      *bool                             `json:"follow_redirects,omitempty"`
		Headers              map[string]string                 `json:"headers,omitempty"`
		Host                 *string                           `json:"host,omitempty"`
		Message              *string                           `json:"message,omitempty"`
		Metadata             map[string]string                 `json:"metadata,omitempty"`
		Method               *HTTPMethod                       `json:"method,omitempty"`
		NoSavingResponseBody *bool                             `json:"noSavingResponseBody,omitempty"`
		NumberOfPackets      *int32                            `json:"numberOfPackets,omitempty"`
		Port                 *int64                            `json:"port,omitempty"`
		Proxy                *SyntheticsTestRequestProxy       `json:"proxy,omitempty"`
		Query                interface{}                       `json:"query,omitempty"`
		Servername           *string                           `json:"servername,omitempty"`
		Service              *string                           `json:"service,omitempty"`
		ShouldTrackHops      *bool                             `json:"shouldTrackHops,omitempty"`
		Timeout              *float64                          `json:"timeout,omitempty"`
		Url                  *string                           `json:"url,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Method; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AllowInsecure = all.AllowInsecure
	o.BasicAuth = all.BasicAuth
	o.Body = all.Body
	if all.Certificate != nil && all.Certificate.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Certificate = all.Certificate
	o.DnsServer = all.DnsServer
	o.DnsServerPort = all.DnsServerPort
	o.FollowRedirects = all.FollowRedirects
	o.Headers = all.Headers
	o.Host = all.Host
	o.Message = all.Message
	o.Metadata = all.Metadata
	o.Method = all.Method
	o.NoSavingResponseBody = all.NoSavingResponseBody
	o.NumberOfPackets = all.NumberOfPackets
	o.Port = all.Port
	if all.Proxy != nil && all.Proxy.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Proxy = all.Proxy
	o.Query = all.Query
	o.Servername = all.Servername
	o.Service = all.Service
	o.ShouldTrackHops = all.ShouldTrackHops
	o.Timeout = all.Timeout
	o.Url = all.Url
	return nil
}
