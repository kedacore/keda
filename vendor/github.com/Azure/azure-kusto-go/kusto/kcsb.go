package kusto

import (
	"fmt"
	kustoErrors "github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"strconv"
	"strings"
)

type ConnectionStringBuilder struct {
	DataSource                       string
	AadUserID                        string
	Password                         string
	UserToken                        string
	ApplicationClientId              string
	ApplicationKey                   string
	AuthorityId                      string
	ApplicationCertificate           string
	ApplicationCertificateThumbprint string
	SendCertificateChain             bool
	ApplicationToken                 string
	AzCli                            bool
	MsiAuthentication                bool
	ManagedServiceIdentity           string
	InteractiveLogin                 bool
	RedirectURL                      string
	DefaultAuth                      bool
	ClientOptions                    *azcore.ClientOptions
}

// params mapping
const (
	dataSource                       string = "DataSource"
	aadUserId                        string = "AADUserID"
	password                         string = "Password"
	applicationClientId              string = "ApplicationClientId"
	applicationKey                   string = "ApplicationKey"
	applicationCertificate           string = "ApplicationCertificate"
	authorityId                      string = "AuthorityId"
	applicationToken                 string = "ApplicationToken"
	userToken                        string = "UserToken"
	applicationCertificateThumbprint string = "ApplicationCertificateThumbprint"
	sendCertificateChain             string = "SendCertificateChain"
	msiAuth                          string = "MSIAuthentication"
	managedServiceIdentity           string = "ManagedServiceIdentity"
	azCli                            string = "AZCLI"
	interactiveLogin                 string = "InteractiveLogin"
	domainHint                       string = "RedirectURL"
)

const (
	BEARER_TYPE = "Bearer"
)

var csMapping = map[string]string{"datasource": dataSource, "data source": dataSource, "addr": dataSource, "address": dataSource, "network address": dataSource, "server": dataSource,
	"aad user id": aadUserId, "aaduserid": aadUserId,
	"password": password, "pwd": password,
	"application client id": applicationClientId, "applicationclientid": applicationClientId, "appclientid": applicationClientId,
	"application key": applicationKey, "applicationkey": applicationKey, "appkey": applicationKey,
	"application certificate": applicationCertificate, "applicationcertificate": applicationCertificate,
	"application certificate thumbprint": applicationCertificateThumbprint, "applicationcertificatethumbprint": applicationCertificateThumbprint,
	"sendcertificatechain": sendCertificateChain, "send certificate chain": sendCertificateChain,
	"authority id": authorityId, "authorityid": authorityId, "authority": authorityId, "tenantid": authorityId, "tenant": authorityId, "tid": authorityId,
	"application token": applicationToken, "applicationtoken": applicationToken, "apptoken": applicationToken,
	"user token": userToken, "usertoken": userToken, "usrtoken": userToken,
	"interactive login": interactiveLogin, "interactivelogin": interactiveLogin,
	"domain hint": domainHint, "domainhint": domainHint,
}

func requireNonEmpty(key string, value string) {
	if isEmpty(value) {
		panic(fmt.Sprintf("Error: %s cannot be null", key))
	}
}

func assignValue(kcsb *ConnectionStringBuilder, rawKey string, value string) error {
	rawKey = strings.ToLower(strings.Trim(rawKey, " "))
	parsedKey, ok := csMapping[rawKey]
	if !ok {
		return fmt.Errorf("Error: unsupported key %q in connection string ", rawKey)
	}
	switch parsedKey {
	case dataSource:
		kcsb.DataSource = value
	case aadUserId:
		kcsb.AadUserID = value
	case password:
		kcsb.Password = value
	case applicationClientId:
		kcsb.ApplicationClientId = value
	case applicationKey:
		kcsb.ApplicationKey = value
	case applicationCertificate:
		kcsb.ApplicationCertificate = value
	case applicationCertificateThumbprint:
		kcsb.ApplicationCertificateThumbprint = value
	case sendCertificateChain:
		bval, _ := strconv.ParseBool(value)
		kcsb.SendCertificateChain = bval
	case authorityId:
		kcsb.AuthorityId = value
	case applicationToken:
		kcsb.ApplicationToken = value
	case userToken:
		kcsb.UserToken = value
	case interactiveLogin:
		bval, _ := strconv.ParseBool(value)
		kcsb.InteractiveLogin = bval
	case domainHint:
		kcsb.RedirectURL = value
	}
	return nil
}

// NewConnectionStringBuilder Creates new Kusto ConnectionStringBuilder.
// Params takes kusto connection string connStr: string.  Kusto connection string should be of the format:
// https://<clusterName>.kusto.windows.net;AAD User ID="user@microsoft.com";Password=P@ssWord
// For more information please look at:
// https://docs.microsoft.com/azure/data-explorer/kusto/api/connection-strings/kusto
func NewConnectionStringBuilder(connStr string) *ConnectionStringBuilder {
	kcsb := ConnectionStringBuilder{}
	if isEmpty(connStr) {
		panic("error: Connection string cannot be empty")
	}
	connStrArr := strings.Split(connStr, ";")
	if !strings.Contains(connStrArr[0], "=") {
		connStrArr[0] = "Data Source=" + connStrArr[0]
	}

	for _, kvp := range connStrArr {
		if isEmpty(strings.Trim(kvp, " ")) {
			continue
		}
		kvparr := strings.Split(kvp, "=")
		val := strings.Trim(kvparr[1], " ")
		if isEmpty(val) {
			continue
		}
		if err := assignValue(&kcsb, kvparr[0], val); err != nil {
			panic(err)
		}
	}
	return &kcsb
}

func (kcsb *ConnectionStringBuilder) resetConnectionString() {
	kcsb.AadUserID = ""
	kcsb.Password = ""
	kcsb.UserToken = ""
	kcsb.ApplicationClientId = ""
	kcsb.ApplicationKey = ""
	kcsb.AuthorityId = ""
	kcsb.ApplicationCertificate = ""
	kcsb.ApplicationCertificateThumbprint = ""
	kcsb.SendCertificateChain = false
	kcsb.ApplicationToken = ""
	kcsb.AzCli = false
	kcsb.MsiAuthentication = false
	kcsb.ManagedServiceIdentity = ""
	kcsb.InteractiveLogin = false
	kcsb.RedirectURL = ""
	kcsb.ClientOptions = nil
	kcsb.DefaultAuth = false
}

// WithAadUserPassAuth Creates a Kusto Connection string builder that will authenticate with AAD user name and password.
func (kcsb *ConnectionStringBuilder) WithAadUserPassAuth(uname string, pswrd string, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	requireNonEmpty(aadUserId, uname)
	requireNonEmpty(password, pswrd)
	kcsb.resetConnectionString()
	kcsb.AadUserID = uname
	kcsb.Password = pswrd
	kcsb.AuthorityId = authorityID
	return kcsb
}

// WitAadUserToken Creates a Kusto Connection string builder that will authenticate with AAD user token
func (kcsb *ConnectionStringBuilder) WitAadUserToken(usertoken string) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	requireNonEmpty(userToken, usertoken)
	kcsb.resetConnectionString()
	kcsb.UserToken = usertoken
	return kcsb
}

// WithAadAppKey Creates a Kusto Connection string builder that will authenticate with AAD application and key.
func (kcsb *ConnectionStringBuilder) WithAadAppKey(appId string, appKey string, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	requireNonEmpty(applicationClientId, appId)
	requireNonEmpty(applicationKey, appKey)
	requireNonEmpty(authorityId, authorityID)
	kcsb.resetConnectionString()
	kcsb.ApplicationClientId = appId
	kcsb.ApplicationKey = appKey
	kcsb.AuthorityId = authorityID
	return kcsb
}

// WithAppCertificate Creates a Kusto Connection string builder that will authenticate with AAD application using a certificate.
func (kcsb *ConnectionStringBuilder) WithAppCertificate(appId string, certificate string, thumprint string, sendCertChain bool, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	requireNonEmpty(applicationCertificate, certificate)
	requireNonEmpty(authorityId, authorityID)
	kcsb.resetConnectionString()
	kcsb.ApplicationClientId = appId
	kcsb.AuthorityId = authorityID

	kcsb.ApplicationCertificate = certificate
	kcsb.ApplicationCertificateThumbprint = thumprint
	kcsb.SendCertificateChain = sendCertChain
	return kcsb
}

// WithApplicationToken Creates a Kusto Connection string builder that will authenticate with AAD application and an application token.
func (kcsb *ConnectionStringBuilder) WithApplicationToken(appId string, appToken string) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	requireNonEmpty(applicationToken, appToken)
	kcsb.resetConnectionString()
	kcsb.ApplicationToken = appToken
	return kcsb
}

// WithAzCli Creates a Kusto Connection string builder that will use existing authenticated az cli profile password.
func (kcsb *ConnectionStringBuilder) WithAzCli() *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.AzCli = true
	return kcsb
}

// WithUserManagedIdentity Creates a Kusto Connection string builder that will authenticate with AAD application, using
// an application token obtained from a Microsoft Service Identity endpoint using user assigned id.
func (kcsb *ConnectionStringBuilder) WithUserManagedIdentity(clientID string) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.MsiAuthentication = true
	kcsb.ManagedServiceIdentity = clientID
	return kcsb
}

// WithSystemManagedIdentity Creates a Kusto Connection string builder that will authenticate with AAD application, using
// an application token obtained from a Microsoft Service Identity endpoint using system assigned id.
func (kcsb *ConnectionStringBuilder) WithSystemManagedIdentity() *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.MsiAuthentication = true
	return kcsb
}

// WithInteractiveLogin Creates a Kusto Connection string builder that will authenticate by launching the system default browser
// to interactively authenticate a user, and obtain an access token
func (kcsb *ConnectionStringBuilder) WithInteractiveLogin(authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	if !isEmpty(authorityID) {
		kcsb.AuthorityId = authorityID
	}
	kcsb.InteractiveLogin = true
	return kcsb
}

// AttachPolicyClientOptions Assigns ClientOptions to string builder that contains configuration settings like Logging and Retry configs for a client's pipeline.
// Read more at https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore@v1.2.0/policy#ClientOptions
func (kcsb *ConnectionStringBuilder) AttachPolicyClientOptions(options *azcore.ClientOptions) *ConnectionStringBuilder {
	requireNonEmpty(dataSource, kcsb.DataSource)
	if options == nil {
		kcsb.ClientOptions = options
	}
	return kcsb
}

// WithDefaultAzureCredential Create Kusto Conntection String that will be used for default auth mode. The order of auth will be via environment variables, managed identity and Azure CLI .
// Read more at https://learn.microsoft.com/azure/developer/go/azure-sdk-authentication?tabs=bash#2-authenticate-with-azure
func (kcsb *ConnectionStringBuilder) WithDefaultAzureCredential() *ConnectionStringBuilder {
	kcsb.resetConnectionString()
	kcsb.DefaultAuth = true
	return kcsb
}

// Method to be used for generating TokenCredential
func (kcsb *ConnectionStringBuilder) newTokenProvider() (*TokenProvider, error) {
	tkp := &TokenProvider{}
	tkp.tokenScheme = BEARER_TYPE

	var init func(*CloudInfo, *azcore.ClientOptions, string) (azcore.TokenCredential, error)

	switch {
	case kcsb.InteractiveLogin:
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			inOpts := &azidentity.InteractiveBrowserCredentialOptions{}
			inOpts.ClientID = ci.KustoClientAppID
			inOpts.TenantID = kcsb.AuthorityId
			inOpts.RedirectURL = ci.KustoClientRedirectURI
			inOpts.ClientOptions = *cliOpts

			cred, err := azidentity.NewInteractiveBrowserCredential(inOpts)
			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Interactive Login. "+
						"Error: %s", err))
			}

			return cred, nil
		}
	case !isEmpty(kcsb.AadUserID) && !isEmpty(kcsb.Password):
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			opts := &azidentity.UsernamePasswordCredentialOptions{ClientOptions: *cliOpts}

			cred, err := azidentity.NewUsernamePasswordCredential(kcsb.AuthorityId, appClientId, kcsb.AadUserID, kcsb.Password, opts)

			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Username Password. Error: %s", err))
			}

			return cred, nil
		}
	case !isEmpty(kcsb.ApplicationClientId) && !isEmpty(kcsb.ApplicationKey):
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			authorityId := kcsb.AuthorityId

			if isEmpty(authorityId) {
				authorityId = ci.FirstPartyAuthorityURL
			}

			opts := &azidentity.ClientSecretCredentialOptions{ClientOptions: *cliOpts}

			cred, err := azidentity.NewClientSecretCredential(authorityId, appClientId, kcsb.ApplicationKey, opts)

			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Client Secret. Error: %s", err))
			}

			return cred, nil
		}
	case !isEmpty(kcsb.ApplicationCertificate):
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			opts := &azidentity.ClientCertificateCredentialOptions{ClientOptions: *cliOpts}
			opts.SendCertificateChain = kcsb.SendCertificateChain

			cert, thumprintKey, err := azidentity.ParseCertificates([]byte(kcsb.ApplicationCertificate), []byte(kcsb.ApplicationCertificateThumbprint))
			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther, err)
			}
			cred, err := azidentity.NewClientCertificateCredential(kcsb.AuthorityId, appClientId, cert, thumprintKey, opts)

			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Application Certificate: %s", err))
			}

			return cred, nil
		}
	case kcsb.MsiAuthentication:
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			opts := &azidentity.ManagedIdentityCredentialOptions{ClientOptions: *cliOpts}
			if !isEmpty(kcsb.ManagedServiceIdentity) {
				opts.ID = azidentity.ClientID(kcsb.ManagedServiceIdentity)
			}

			cred, err := azidentity.NewManagedIdentityCredential(opts)

			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Managed Identity: %s", err))
			}

			return cred, nil
		}
	case !isEmpty(kcsb.UserToken):
		{
			tkp.customToken = kcsb.UserToken
		}
	case !isEmpty(kcsb.ApplicationToken):
		{
			tkp.customToken = kcsb.ApplicationToken
		}
	case kcsb.AzCli:
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			authorityId := kcsb.AuthorityId

			if isEmpty(authorityId) {
				authorityId = ci.FirstPartyAuthorityURL
			}

			opts := &azidentity.AzureCLICredentialOptions{}
			opts.TenantID = kcsb.AuthorityId
			cred, err := azidentity.NewAzureCLICredential(opts)

			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Azure CLI: %s", err))
			}

			return cred, nil
		}
	case kcsb.DefaultAuth:
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			//Default Azure authentication
			opts := &azidentity.DefaultAzureCredentialOptions{}
			if kcsb.ClientOptions != nil {
				opts.ClientOptions = *cliOpts
			}
			if !isEmpty(kcsb.AuthorityId) {
				opts.TenantID = kcsb.AuthorityId
			}

			cred, err := azidentity.NewDefaultAzureCredential(opts)

			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials for DefaultAzureCredential: %s", err))
			}

			return cred, nil
		}
	}

	if init != nil {
		tkp.setInit(kcsb, init)
	}

	return tkp, nil
}

func isEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}
