package azkustodata

import (
	"encoding/base64"
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/keywords"
	"os"
	"strconv"
	"strings"

	kustoErrors "github.com/Azure/azure-kusto-go/azkustodata/errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type ConnectionStringBuilder struct {
	DataSource                       string
	InitialCatalog                   string // TODO - implement default db support
	AadFederatedSecurity             bool
	AadUserID                        string
	Password                         string
	UserToken                        string
	ApplicationClientId              string
	ApplicationKey                   string
	AuthorityId                      string
	ApplicationCertificatePath       string
	ApplicationCertificateBytes      []byte
	ApplicationCertificatePassword   []byte
	SendCertificateChain             bool
	ApplicationToken                 string
	AzCli                            bool
	MsiAuthentication                bool
	WorkloadAuthentication           bool
	FederationTokenFilePath          string
	ManagedServiceIdentityClientId   string
	ManagedServiceIdentityResourceId string
	InteractiveLogin                 bool
	RedirectURL                      string
	DefaultAuth                      bool
	ClientOptions                    *azcore.ClientOptions
	ApplicationForTracing            string
	UserForTracing                   string
	TokenCredential                  azcore.TokenCredential
}

const (
	BearerType        = "Bearer"
	SecretReplacement = "****"
)

func requireNonEmpty(key string, value string) {
	if isEmpty(value) {
		panic(fmt.Sprintf("Error: %s cannot be null", key))
	}
}

func assignValue(kcsb *ConnectionStringBuilder, rawKey string, value string) error {
	keyword, err := keywords.GetKeyword(rawKey)
	if err != nil {
		return err
	}

	switch keyword.Name {
	case keywords.DataSource:
		kcsb.DataSource = value
	case keywords.InitialCatalog:
		kcsb.InitialCatalog = value
	case keywords.FederatedSecurity:
		bval, err := strconv.ParseBool(value)
		if err != nil {
			return kustoErrors.ES(kustoErrors.OpUnknown, kustoErrors.KOther, "error: Couldn't parse federated security value: %s", err)
		}
		kcsb.AadFederatedSecurity = bval
	case keywords.ApplicationClientId:
		kcsb.ApplicationClientId = value
	case keywords.UserId:
		kcsb.AadUserID = value
	case keywords.AuthorityId:
		kcsb.AuthorityId = value

	case keywords.ApplicationToken:
		kcsb.ApplicationToken = value
	case keywords.UserToken:
		kcsb.UserToken = value
	case keywords.ApplicationKey:
		kcsb.ApplicationKey = value
	case keywords.ApplicationCertificateX5C:
		bval, err := strconv.ParseBool(value)
		if err != nil {
			return kustoErrors.ES(kustoErrors.OpUnknown, kustoErrors.KOther, "error: Couldn't parse certificate x5c value: %s", err)
		}
		kcsb.SendCertificateChain = bval
	case keywords.ApplicationCertificateBlob:
		decodeString, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return kustoErrors.ES(kustoErrors.OpUnknown, kustoErrors.KOther, "error: Couldn't decode certificate blob: %s", err)
		}
		kcsb.ApplicationCertificateBytes = decodeString
	case keywords.ApplicationNameForTracing:
		kcsb.ApplicationForTracing = value
	case keywords.UserNameForTracing:
		kcsb.UserForTracing = value
	case keywords.Password:
		kcsb.Password = value
	}

	return nil
}

// ConnectionString Generates a connection string from the current state of the ConnectionStringBuilder.
func (kcsb *ConnectionStringBuilder) ConnectionString(includeSecrets bool) (string, error) {
	builder := strings.Builder{}

	writeValue := func(k string, v string) error {
		if isEmpty(v) {
			return nil
		}

		keyword, err := keywords.GetKeyword(k)
		if err != nil {
			return err
		}

		builder.WriteString(keyword.Name)
		builder.WriteRune('=')
		if keyword.Secret && !includeSecrets {
			builder.WriteString(SecretReplacement)
		} else {
			builder.WriteString(v)
		}

		builder.WriteRune(';')
		return nil
	}

	if err := writeValue(keywords.DataSource, kcsb.DataSource); err != nil {
		return "", err
	}
	if err := writeValue(keywords.InitialCatalog, kcsb.InitialCatalog); err != nil {
		return "", err
	}
	if kcsb.AadFederatedSecurity {
		if err := writeValue(keywords.FederatedSecurity, "true"); err != nil {
			return "", err
		}
	}
	if err := writeValue(keywords.ApplicationClientId, kcsb.ApplicationClientId); err != nil {
		return "", err
	}
	if err := writeValue(keywords.UserId, kcsb.AadUserID); err != nil {
		return "", err
	}
	if err := writeValue(keywords.AuthorityId, kcsb.AuthorityId); err != nil {
		return "", err
	}
	if err := writeValue(keywords.ApplicationToken, kcsb.ApplicationToken); err != nil {
		return "", err
	}
	if err := writeValue(keywords.UserToken, kcsb.UserToken); err != nil {
		return "", err
	}
	if err := writeValue(keywords.ApplicationKey, kcsb.ApplicationKey); err != nil {
		return "", err
	}
	if kcsb.SendCertificateChain {
		if err := writeValue(keywords.ApplicationCertificateX5C, "true"); err != nil {
			return "", err
		}
	}
	if len(kcsb.ApplicationCertificateBytes) != 0 {
		if err := writeValue(keywords.ApplicationCertificateBlob, base64.StdEncoding.EncodeToString(kcsb.ApplicationCertificateBytes)); err != nil {
			return "", err
		}
	}

	s := builder.String()
	if len(s) > 0 {
		s = s[:len(s)-1] // remove trailing ';'
	}

	return s, nil
}

// NewConnectionStringBuilder Creates new Kusto ConnectionStringBuilder.
// Params takes kusto connection string connStr: string.  Kusto connection string should be of the format:
// https://<clusterName>.<location>.kusto.windows.net;AAD User ID="user@microsoft.com";Password=P@ssWord
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
	kcsb.AadFederatedSecurity = false
	kcsb.AadUserID = ""
	kcsb.InitialCatalog = ""
	kcsb.Password = ""
	kcsb.UserToken = ""
	kcsb.ApplicationClientId = ""
	kcsb.ApplicationKey = ""
	kcsb.AuthorityId = ""
	kcsb.ApplicationCertificatePath = ""
	kcsb.ApplicationCertificateBytes = nil
	kcsb.ApplicationCertificatePassword = nil
	kcsb.SendCertificateChain = false
	kcsb.ApplicationToken = ""
	kcsb.AzCli = false
	kcsb.MsiAuthentication = false
	kcsb.WorkloadAuthentication = false
	kcsb.ManagedServiceIdentityClientId = ""
	kcsb.ManagedServiceIdentityResourceId = ""
	kcsb.InteractiveLogin = false
	kcsb.RedirectURL = ""
	kcsb.ClientOptions = nil
	kcsb.DefaultAuth = false
	kcsb.TokenCredential = nil
}

// WithAadUserPassAuth Creates a Kusto Connection string builder that will authenticate with AAD user name and password.
func (kcsb *ConnectionStringBuilder) WithAadUserPassAuth(uname string, pswrd string, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	requireNonEmpty(keywords.UserId, uname)
	requireNonEmpty(keywords.Password, pswrd)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.AadUserID = uname
	kcsb.Password = pswrd
	kcsb.AuthorityId = authorityID
	return kcsb
}

// WithAadUserToken Creates a Kusto Connection string builder that will authenticate with AAD user token
func (kcsb *ConnectionStringBuilder) WithAadUserToken(usertoken string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	requireNonEmpty(keywords.UserToken, usertoken)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.UserToken = usertoken
	return kcsb
}

// WithAadAppKey Creates a Kusto Connection string builder that will authenticate with AAD application and key.
func (kcsb *ConnectionStringBuilder) WithAadAppKey(appId string, appKey string, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	requireNonEmpty(keywords.ApplicationClientId, appId)
	requireNonEmpty(keywords.ApplicationKey, appKey)
	requireNonEmpty(keywords.AuthorityId, authorityID)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.ApplicationClientId = appId
	kcsb.ApplicationKey = appKey
	kcsb.AuthorityId = authorityID
	return kcsb
}

// WithAppCertificatePath Creates a Kusto Connection string builder that will authenticate with AAD application using a certificate.
func (kcsb *ConnectionStringBuilder) WithAppCertificatePath(appId string, certificatePath string, password []byte, sendCertChain bool, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	requireNonEmpty("Application Certificate Path", certificatePath)
	requireNonEmpty(keywords.AuthorityId, authorityID)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.ApplicationClientId = appId
	kcsb.AuthorityId = authorityID

	kcsb.ApplicationCertificatePath = certificatePath
	kcsb.ApplicationCertificatePassword = password
	kcsb.SendCertificateChain = sendCertChain
	return kcsb
}

// WithAppCertificateBytes Creates a Kusto Connection string builder that will authenticate with AAD application using a certificate.
func (kcsb *ConnectionStringBuilder) WithAppCertificateBytes(appId string, certificateBytes []byte, password []byte, sendCertChain bool, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	requireNonEmpty(keywords.AuthorityId, authorityID)
	if len(certificateBytes) == 0 {
		panic("error: Certificate cannot be null")
	}
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.ApplicationClientId = appId
	kcsb.AuthorityId = authorityID

	kcsb.ApplicationCertificateBytes = certificateBytes
	kcsb.ApplicationCertificatePassword = password
	kcsb.SendCertificateChain = sendCertChain
	return kcsb
}

// WithApplicationToken Creates a Kusto Connection string builder that will authenticate with AAD application and an application token.
func (kcsb *ConnectionStringBuilder) WithApplicationToken(appId string, appToken string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	requireNonEmpty(keywords.ApplicationToken, appToken)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.ApplicationToken = appToken
	return kcsb
}

// WithAzCli Creates a Kusto Connection string builder that will use existing authenticated az cli profile password.
func (kcsb *ConnectionStringBuilder) WithAzCli() *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.AzCli = true
	return kcsb
}

// WithUserAssignedIdentityClientId Creates a Kusto Connection string builder that will authenticate with AAD application, using
// an application token obtained from a Microsoft Service Identity endpoint using user assigned id.
func (kcsb *ConnectionStringBuilder) WithUserAssignedIdentityClientId(clientID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.MsiAuthentication = true
	kcsb.ManagedServiceIdentityClientId = clientID
	return kcsb
}

// WithUserAssignedIdentityResourceId Creates a Kusto Connection string builder that will authenticate with AAD application, using
// an application token obtained from a Microsoft Service Identity endpoint using an MSI's resourceID.
func (kcsb *ConnectionStringBuilder) WithUserAssignedIdentityResourceId(resourceID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.MsiAuthentication = true
	kcsb.ManagedServiceIdentityResourceId = resourceID
	return kcsb
}

// WithSystemManagedIdentity Creates a Kusto Connection string builder that will authenticate with AAD application, using
// an application token obtained from a Microsoft Service Identity endpoint using system assigned id.
func (kcsb *ConnectionStringBuilder) WithSystemManagedIdentity() *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.MsiAuthentication = true
	return kcsb
}

// WithKubernetesWorkloadIdentity Creates a Kusto Connection string builder that will authenticate with AAD application, using
// an application token obtained from a Microsoft Service Identity endpoint using Kubernetes workload identity.
func (kcsb *ConnectionStringBuilder) WithKubernetesWorkloadIdentity(appId, tokenFilePath, authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.ApplicationClientId = appId
	kcsb.AuthorityId = authorityID
	kcsb.FederationTokenFilePath = tokenFilePath
	kcsb.WorkloadAuthentication = true
	return kcsb
}

// WithInteractiveLogin Creates a Kusto Connection string builder that will authenticate by launching the system default browser
// to interactively authenticate a user, and obtain an access token
func (kcsb *ConnectionStringBuilder) WithInteractiveLogin(authorityID string) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	if !isEmpty(authorityID) {
		kcsb.AuthorityId = authorityID
	}
	kcsb.InteractiveLogin = true
	return kcsb
}

// AttachPolicyClientOptions Assigns ClientOptions to string builder that contains configuration settings like Logging and Retry configs for a client's pipeline.
// Read more at https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore@v1.2.0/policy#ClientOptions
func (kcsb *ConnectionStringBuilder) AttachPolicyClientOptions(options *azcore.ClientOptions) *ConnectionStringBuilder {
	requireNonEmpty(keywords.DataSource, kcsb.DataSource)
	if options != nil {
		kcsb.ClientOptions = options
	}
	return kcsb
}

// WithDefaultAzureCredential Create Kusto Conntection String that will be used for default auth mode. The order of auth will be via environment variables, managed identity and Azure CLI .
// Read more at https://learn.microsoft.com/azure/developer/go/azure-sdk-authentication?tabs=bash#2-authenticate-with-azure
func (kcsb *ConnectionStringBuilder) WithDefaultAzureCredential() *ConnectionStringBuilder {
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.DefaultAuth = true
	return kcsb
}

func (kcsb *ConnectionStringBuilder) WithTokenCredential(tokenCredential azcore.TokenCredential) *ConnectionStringBuilder {
	kcsb.resetConnectionString()
	kcsb.AadFederatedSecurity = true
	kcsb.TokenCredential = tokenCredential
	return kcsb
}

// Method to be used for generating TokenCredential
func (kcsb *ConnectionStringBuilder) newTokenProvider() (*TokenProvider, error) {
	tkp := &TokenProvider{}
	tkp.tokenScheme = BearerType

	var init func(*CloudInfo, *azcore.ClientOptions, string) (azcore.TokenCredential, error)

	switch {
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
	case !isEmpty(kcsb.ApplicationCertificatePath) || len(kcsb.ApplicationCertificateBytes) != 0:
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			opts := &azidentity.ClientCertificateCredentialOptions{ClientOptions: *cliOpts}
			opts.SendCertificateChain = kcsb.SendCertificateChain

			bytes := kcsb.ApplicationCertificateBytes
			if !isEmpty(kcsb.ApplicationCertificatePath) {
				// read the certificate from the file
				fileBytes, err := os.ReadFile(kcsb.ApplicationCertificatePath)
				if err != nil {
					return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
						fmt.Errorf("error: Couldn't read certificate file: %s", err))
				}
				bytes = fileBytes
			}

			cert, thumprintKey, err := azidentity.ParseCertificates(bytes, kcsb.ApplicationCertificatePassword)
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
			// legacy kcsb.ManagedServiceIdentity field takes precedence over
			// new kcsb.ManagedServiceIdentityClientId field which takes precedence over
			// new kcsb.ManagedServiceIdentityResourceId field
			// if no client id is provided, the logic falls back to set up
			// the system assigned identity
			if !isEmpty(kcsb.ManagedServiceIdentityClientId) {
				opts.ID = azidentity.ClientID(kcsb.ManagedServiceIdentityClientId)
			} else if !isEmpty(kcsb.ManagedServiceIdentityResourceId) {
				opts.ID = azidentity.ResourceID(kcsb.ManagedServiceIdentityResourceId)
			}

			cred, err := azidentity.NewManagedIdentityCredential(opts)

			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Managed Identity: %s", err))
			}

			return cred, nil
		}
	case kcsb.WorkloadAuthentication:
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			opts := &azidentity.WorkloadIdentityCredentialOptions{ClientOptions: *cliOpts}
			if !isEmpty(kcsb.ApplicationClientId) {
				opts.ClientID = kcsb.ApplicationClientId
			}

			if !isEmpty(kcsb.FederationTokenFilePath) {
				opts.TokenFilePath = kcsb.FederationTokenFilePath
			}

			if !isEmpty(kcsb.AuthorityId) {
				opts.TenantID = kcsb.AuthorityId
			}

			cred, err := azidentity.NewWorkloadIdentityCredential(opts)
			if err != nil {
				return nil, kustoErrors.E(kustoErrors.OpTokenProvider, kustoErrors.KOther,
					fmt.Errorf("error: Couldn't retrieve client credentials using Workload Identity: %s", err))
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
			opts.ClientOptions = *cliOpts
			if kcsb.ClientOptions != nil {
				opts.ClientOptions = *kcsb.ClientOptions
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
	case kcsb.TokenCredential != nil:
		init = func(ci *CloudInfo, cliOpts *azcore.ClientOptions, appClientId string) (azcore.TokenCredential, error) {
			return kcsb.TokenCredential, nil
		}
	case kcsb.InteractiveLogin || kcsb.AadFederatedSecurity: // If AadFed is set, but no other auth method is set, default to interactive login
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
	}

	if init != nil {
		tkp.setInit(kcsb, init)
	}

	return tkp, nil
}

func isEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

func (kcsb *ConnectionStringBuilder) SetConnectorDetails(name, version, appName, appVersion string, sendUser bool, overrideUser string, additionalFields ...StringPair) {
	app, user := setConnectorDetails(name, version, appName, appVersion, sendUser, overrideUser, additionalFields...)
	kcsb.ApplicationForTracing = app
	kcsb.UserForTracing = user
}
