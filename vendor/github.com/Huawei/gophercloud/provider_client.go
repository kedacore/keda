package gophercloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	//"strconv"
	"strings"
	"sync"

	"github.com/Huawei/gophercloud/auth/aksk"
)

// DefaultUserAgent is the default User-Agent string set in the request header.
const DefaultUserAgent = "huawei-cloud-sdk-go/1.0.21"

// ProviderClient stores details that are required to interact with any
// services within a specific provider's API.
//
// Generally, you acquire a ProviderClient by calling the NewClient method in
// the appropriate provider's child package, providing whatever authentication
// credentials are required.
type ProviderClient struct {
	// IdentityBase is the base URL used for a particular provider's identity
	// service - it will be used when issuing authenticatation requests. It
	// should point to the root resource of the identity service, not a specific
	// identity version.
	IdentityBase string

	// IdentityEndpoint is the identity endpoint. This may be a specific version
	// of the identity service. If this is the case, this endpoint is used rather
	// than querying versions first.
	IdentityEndpoint string

	// TokenID is the ID of the most recently issued valid token.
	// NOTE: Aside from within a custom ReauthFunc, this field shouldn't be set by an application.
	// To safely read or write this value, call `Token` or `SetToken`, respectively
	TokenID string

	// EndpointLocator describes how this provider discovers the endpoints for
	// its constituent services.
	EndpointLocator EndpointLocator

	// HTTPClient allows users to interject arbitrary http, https, or other transit behaviors.
	HTTPClient http.Client

	// UserAgent represents the User-Agent header in the HTTP request.
	UserAgent UserAgent

	// ReauthFunc is the function used to re-authenticate the user if the request
	// fails with a 401 HTTP response code. This a needed because there may be multiple
	// authentication functions for different Identity service versions.
	ReauthFunc func() error

	// mut is a mutex for the client. It protects read and write access to client attributes such as getting
	// and setting the TokenID.
	mut *sync.RWMutex

	// reauthmut is a mutex for reauthentication it attempts to ensure that only one reauthentication
	// attempt happens at one time.
	reauthmut *reauthlock

	// DomainID
	DomainID string

	// ProjectID
	ProjectID string

	// Conf define the configs parameter of the provider client
	Conf *Config

	// AKSKAuthOptions provides the value for AK/SK authentication, it should be nil if you use token authentication,
	// Otherwise, it must have a value
	AKSKOptions aksk.AKSKOptions
}

// reauthlock represents a set of attributes used to help in the reauthentication process.
type reauthlock struct {
	sync.RWMutex
	reauthing bool
}

//GetProjectID,Implement the GetProjectID() interface, return client projectID.
func (client *ProviderClient) GetProjectID() string {
	return client.ProjectID
}

// AuthenticatedHeaders returns a map of HTTP headers that are common for all
// authenticated service requests.
func (client *ProviderClient) AuthenticatedHeaders() (m map[string]string) {
	if client.reauthmut != nil {
		client.reauthmut.RLock()
		if client.reauthmut.reauthing {
			client.reauthmut.RUnlock()
			return
		}
		client.reauthmut.RUnlock()
	}
	t := client.Token()
	if t == "" {
		return
	}
	return map[string]string{"X-Auth-Token": t}
}

// UseTokenLock creates a mutex that is used to allow safe concurrent access to the auth token.
// If the application's ProviderClient is not used concurrently, this doesn't need to be called.
func (client *ProviderClient) UseTokenLock() {
	client.mut = new(sync.RWMutex)
	client.reauthmut = new(reauthlock)
}

// Token safely reads the value of the auth token from the ProviderClient. Applications should
// call this method to access the token instead of the TokenID field
func (client *ProviderClient) Token() string {
	if client.mut != nil {
		client.mut.RLock()
		defer client.mut.RUnlock()
	}
	return client.TokenID
}

// SetToken safely sets the value of the auth token in the ProviderClient. Applications may
// use this method in a custom ReauthFunc
func (client *ProviderClient) SetToken(t string) {
	if client.mut != nil {
		client.mut.Lock()
		defer client.mut.Unlock()
	}
	client.TokenID = t
}

//Reauthenticate calls client.ReauthFunc in a thread-safe way. If this is
//called because of a 401 response, the caller may pass the previous token. In
//this case, the reauthentication can be skipped if another thread has already
//reauthenticated in the meantime. If no previous token is known, an empty
//string should be passed instead to force unconditional reauthentication.
func (client *ProviderClient) Reauthenticate(previousToken string) (err error) {
	if client.ReauthFunc == nil {
		return nil
	}

	if client.mut == nil {
		return client.ReauthFunc()
	}
	client.mut.Lock()
	defer client.mut.Unlock()

	client.reauthmut.Lock()
	client.reauthmut.reauthing = true
	client.reauthmut.Unlock()

	if previousToken == "" || client.TokenID == previousToken {
		err = client.ReauthFunc()
	}

	client.reauthmut.Lock()
	client.reauthmut.reauthing = false
	client.reauthmut.Unlock()
	return
}

// RequestOpts customizes the behavior of the provider.Request() method.
type RequestOpts struct {
	// JSONBody, if provided, will be encoded as JSON and used as the body of the HTTP request. The
	// content type of the request will default to "application/json" unless overridden by MoreHeaders.
	// It's an error to specify both a JSONBody and a RawBody.
	JSONBody interface{}
	// RawBody contains an io.Reader that will be consumed by the request directly. No content-type
	// will be set unless one is provided explicitly by MoreHeaders.
	RawBody io.Reader
	// JSONResponse, if provided, will be populated with the contents of the response body parsed as
	// JSON.
	JSONResponse interface{}
	// OkCodes contains a list of numeric HTTP status codes that should be interpreted as success. If
	// the response has a different code, an error will be returned.
	OkCodes []int
	// MoreHeaders specifies additional HTTP headers to be provide on the request. If a header is
	// provided with a blank value (""), that header will be *omitted* instead: use this to suppress
	// the default Accept header or an inferred Content-Type, for example.
	MoreHeaders map[string]string
	// ErrorContext specifies the resource error type to return if an error is encountered.
	// This lets resources override default error messages based on the response status code.
	ErrorContext error

	HandleError func(httpStatus int, responseContent string) error
}

var applicationJSON = "application/json"

// Request performs an HTTP request using the ProviderClient's current HTTPClient. An authentication
// header will automatically be provided.
func (client *ProviderClient) Request(method, url string, options *RequestOpts) (*http.Response, error) {
	req, err := buildReq(client, method, url, options)
	if err != nil {
		return nil, err
	}
	log := GetLogger()
	prereqtok := req.Header.Get("X-Auth-Token")
	var resp *http.Response

	/*
		//根据配置执行超时重连
		for retryTimes := 0; retryTimes <= client.Conf.MaxRetryTime; retryTimes++ {
			resp, err = client.HTTPClient.Do(req)

			var timeout bool
			// receive error
			if err != nil {
				if timeout = isTimeout(err); !timeout {
					//fmt.Println("timeout:", timeout)
					// if not timeout error, return
					return nil, err
				} else if retryTimes >= client.Conf.MaxRetryTime {
					timeoutErrorMsg := fmt.Sprintf(CE_TimeoutErrorMessage, strconv.Itoa(retryTimes+1), strconv.Itoa(retryTimes+1))
					err := NewSystemCommonError(CE_TimeoutErrorCode, timeoutErrorMsg)
					return nil, err
				}
			}

			//  if status code >= 500 or timeout, will trigger retry
			if client.Conf.AutoRetry && (timeout || isServerError(resp)) {
				req, err = buildReq(client, method, url, options)
				if err != nil {
					return nil, err
				}

				continue
			}
			break
		}
	*/

	//fmt.Println("url:", url)
	//Issue the request.  原代码，注释掉
	resp, err = client.HTTPClient.Do(req)
	if err != nil {
		log.Debug("Request error", err)
		return nil, err
	}

	log.Debug("Request method is %s,Request url is %s", req.Method, url)
	log.Debug("Request header is %s", req.Header)
	log.Debug("Response status code is %d", resp.StatusCode)
	log.Debug("Response header is %s", resp.Header)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close() //  must close
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	log.Debug("Response body is %s\n", string(bodyBytes))

	// Allow default OkCodes if none explicitly set
	if options.OkCodes == nil {
		options.OkCodes = defaultOkCodes(method)
	}

	// Validate the HTTP response status.
	var ok bool
	for _, code := range options.OkCodes {
		if resp.StatusCode == code {
			ok = true
			break
		}
	}
	if !ok {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		log.Debug("Request failed response body is %s", string(body))
		/*
		    http.StatusBadRequest: //400
		   	http.StatusUnauthorized: //401
		   	http.StatusForbidden: //403
		   	http.StatusNotFound: //404
		   	http.StatusMethodNotAllowed: //405
		   	http.StatusRequestTimeout: //408
		    http.StatusConflict: //409
		   	http.StatusTooManyRequests: //429
		   	http.StatusInternalServerError: //500
		   	http.StatusServiceUnavailable: //503
		*/
		switch resp.StatusCode {
		case http.StatusUnauthorized: //401
			if client.ReauthFunc != nil {
				return doReauthAndReq(client, prereqtok, method, url, options)
			}
		case http.StatusForbidden:
			b := strings.Contains(string(body), "Token need to refresh")
			if client.ReauthFunc != nil && b {
				return doReauthAndReq(client, prereqtok, method, url, options)
			}
		}

		if options.HandleError != nil {
			return resp, options.HandleError(resp.StatusCode, string(body))
		}

		return resp, NewSystemServerError(resp.StatusCode, string(body))
	}

	// Parse the response body as JSON, if requested to do so.
	if options.JSONResponse != nil {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(options.JSONResponse); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

//构造request对象
func buildReq(client *ProviderClient, method, url string, options *RequestOpts) (*http.Request, error) {
	var body io.Reader
	var contentType *string

	// Derive the content body by either encoding an arbitrary object as JSON, or by taking a provided
	// io.ReadSeeker as-is. Default the content-type to application/json.
	if options.JSONBody != nil {
		if options.RawBody != nil {
			panic("Please provide only one of JSONBody or RawBody to gophercloud.Request().")
		}

		rendered, err := json.Marshal(options.JSONBody)
		if err != nil {
			return nil, err
		}
		GetLogger().Debug("Request body is %s", string(rendered))
		body = bytes.NewReader(rendered)
		contentType = &applicationJSON
	}

	if options.RawBody != nil {
		body = options.RawBody
	}

	// Construct the http.Request.
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Populate the request headers. Apply options.MoreHeaders last, to give the caller the chance to
	// modify or omit any header.
	if contentType != nil {
		req.Header.Set("Content-Type", *contentType)
	}
	req.Header.Set("Accept", applicationJSON)

	// Set the User-Agent header
	req.Header.Set("User-Agent", client.UserAgent.Join())

	if options.MoreHeaders != nil {
		for k, v := range options.MoreHeaders {
			if v != "" {
				req.Header.Set(k, v)
			} else {
				req.Header.Del(k)
			}
		}
	}

	// get latest token from client
	for k, v := range client.AuthenticatedHeaders() {
		req.Header.Set(k, v)
	}

	if client.AKSKOptions.AccessKey != "" {
		aksk.Sign(req, aksk.SignOptions{
			AccessKey: client.AKSKOptions.AccessKey,
			SecretKey: client.AKSKOptions.SecretKey,
		})
	}

	// Set connection parameter to close the connection immediately when we've got the response
	req.Close = true

	return req, nil
}

//reauth and request
func doReauthAndReq(client *ProviderClient, prereqtok, method, url string, options *RequestOpts) (*http.Response, error) {
	err := client.Reauthenticate(prereqtok)
	if err != nil {
		message:=fmt.Sprintf(CE_ReauthFuncErrorMessage, err.Error())
		return nil, NewSystemCommonError(CE_ReauthFuncErrorCode, message)
	}
	if options.RawBody != nil {
		if seeker, ok := options.RawBody.(io.Seeker); ok {
			seeker.Seek(0, 0)
		}
	}
	resp, err := client.Request(method, url, options)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func isTimeout(err error) bool {
	if err == nil {
		return true
	}
	netErr, isNetError := err.(net.Error)
	return isNetError && netErr.Timeout()
}

func isServerError(httpResponse *http.Response) bool {
	return httpResponse.StatusCode >= http.StatusInternalServerError
}

func defaultOkCodes(method string) []int {
	switch {
	case method == "GET":
		return []int{200}
	case method == "POST":
		return []int{201, 202}
	case method == "PUT":
		return []int{201, 202}
	case method == "PATCH":
		return []int{200, 204}
	case method == "DELETE":
		return []int{202, 204}
	}

	return []int{}
}

// UserAgent represents a User-Agent header.
type UserAgent struct {
	// prepend is the slice of User-Agent strings to prepend to DefaultUserAgent.
	// All the strings to prepend are accumulated and prepended in the Join method.
	prepend []string
}

// Prepend prepends a user-defined string to the default User-Agent string. Users
// may pass in one or more strings to prepend.
func (ua *UserAgent) Prepend(s ...string) {
	ua.prepend = append(s, ua.prepend...)
}

// Join concatenates all the user-defined User-Agend strings with the default
// Gophercloud User-Agent string.
func (ua *UserAgent) Join() string {
	uaSlice := append(ua.prepend, DefaultUserAgent)
	return strings.Join(uaSlice, " ")
}
