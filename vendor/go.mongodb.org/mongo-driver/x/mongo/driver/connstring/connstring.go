// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package connstring // import "go.mongodb.org/mongo-driver/x/mongo/driver/connstring"

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/internal/randutil"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/mongo/driver/dns"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

const (
	// ServerMonitoringModeAuto indicates that the client will behave like "poll"
	// mode when running on a FaaS (Function as a Service) platform, or like
	// "stream" mode otherwise. The client detects its execution environment by
	// following the rules for generating the "client.env" handshake metadata field
	// as specified in the MongoDB Handshake specification. This is the default
	// mode.
	ServerMonitoringModeAuto = "auto"

	// ServerMonitoringModePoll indicates that the client will periodically check
	// the server using a hello or legacy hello command and then sleep for
	// heartbeatFrequencyMS milliseconds before running another check.
	ServerMonitoringModePoll = "poll"

	// ServerMonitoringModeStream indicates that the client will use a streaming
	// protocol when the server supports it. The streaming protocol optimally
	// reduces the time it takes for a client to discover server state changes.
	ServerMonitoringModeStream = "stream"
)

var (
	// ErrLoadBalancedWithMultipleHosts is returned when loadBalanced=true is
	// specified in a URI with multiple hosts.
	ErrLoadBalancedWithMultipleHosts = errors.New(
		"loadBalanced cannot be set to true if multiple hosts are specified")

	// ErrLoadBalancedWithReplicaSet is returned when loadBalanced=true is
	// specified in a URI with the replicaSet option.
	ErrLoadBalancedWithReplicaSet = errors.New(
		"loadBalanced cannot be set to true if a replica set name is specified")

	// ErrLoadBalancedWithDirectConnection is returned when loadBalanced=true is
	// specified in a URI with the directConnection option.
	ErrLoadBalancedWithDirectConnection = errors.New(
		"loadBalanced cannot be set to true if the direct connection option is specified")

	// ErrSRVMaxHostsWithReplicaSet is returned when srvMaxHosts > 0 is
	// specified in a URI with the replicaSet option.
	ErrSRVMaxHostsWithReplicaSet = errors.New(
		"srvMaxHosts cannot be a positive value if a replica set name is specified")

	// ErrSRVMaxHostsWithLoadBalanced is returned when srvMaxHosts > 0 is
	// specified in a URI with loadBalanced=true.
	ErrSRVMaxHostsWithLoadBalanced = errors.New(
		"srvMaxHosts cannot be a positive value if loadBalanced is set to true")
)

// random is a package-global pseudo-random number generator.
var random = randutil.NewLockedRand()

// ParseAndValidate parses the provided URI into a ConnString object.
// It check that all values are valid.
func ParseAndValidate(s string) (ConnString, error) {
	p := parser{dnsResolver: dns.DefaultResolver}
	err := p.parse(s)
	if err != nil {
		return p.ConnString, fmt.Errorf("error parsing uri: %w", err)
	}
	err = p.ConnString.Validate()
	if err != nil {
		return p.ConnString, fmt.Errorf("error validating uri: %w", err)
	}
	return p.ConnString, nil
}

// Parse parses the provided URI into a ConnString object
// but does not check that all values are valid. Use `ConnString.Validate()`
// to run the validation checks separately.
func Parse(s string) (ConnString, error) {
	p := parser{dnsResolver: dns.DefaultResolver}
	err := p.parse(s)
	if err != nil {
		err = fmt.Errorf("error parsing uri: %w", err)
	}
	return p.ConnString, err
}

// ConnString represents a connection string to mongodb.
type ConnString struct {
	Original                           string
	AppName                            string
	AuthMechanism                      string
	AuthMechanismProperties            map[string]string
	AuthMechanismPropertiesSet         bool
	AuthSource                         string
	AuthSourceSet                      bool
	Compressors                        []string
	Connect                            ConnectMode
	ConnectSet                         bool
	DirectConnection                   bool
	DirectConnectionSet                bool
	ConnectTimeout                     time.Duration
	ConnectTimeoutSet                  bool
	Database                           string
	HeartbeatInterval                  time.Duration
	HeartbeatIntervalSet               bool
	Hosts                              []string
	J                                  bool
	JSet                               bool
	LoadBalanced                       bool
	LoadBalancedSet                    bool
	LocalThreshold                     time.Duration
	LocalThresholdSet                  bool
	MaxConnIdleTime                    time.Duration
	MaxConnIdleTimeSet                 bool
	MaxPoolSize                        uint64
	MaxPoolSizeSet                     bool
	MinPoolSize                        uint64
	MinPoolSizeSet                     bool
	MaxConnecting                      uint64
	MaxConnectingSet                   bool
	Password                           string
	PasswordSet                        bool
	ReadConcernLevel                   string
	ReadPreference                     string
	ReadPreferenceTagSets              []map[string]string
	RetryWrites                        bool
	RetryWritesSet                     bool
	RetryReads                         bool
	RetryReadsSet                      bool
	MaxStaleness                       time.Duration
	MaxStalenessSet                    bool
	ReplicaSet                         string
	Scheme                             string
	ServerMonitoringMode               string
	ServerSelectionTimeout             time.Duration
	ServerSelectionTimeoutSet          bool
	SocketTimeout                      time.Duration
	SocketTimeoutSet                   bool
	SRVMaxHosts                        int
	SRVServiceName                     string
	SSL                                bool
	SSLSet                             bool
	SSLClientCertificateKeyFile        string
	SSLClientCertificateKeyFileSet     bool
	SSLClientCertificateKeyPassword    func() string
	SSLClientCertificateKeyPasswordSet bool
	SSLCertificateFile                 string
	SSLCertificateFileSet              bool
	SSLPrivateKeyFile                  string
	SSLPrivateKeyFileSet               bool
	SSLInsecure                        bool
	SSLInsecureSet                     bool
	SSLCaFile                          string
	SSLCaFileSet                       bool
	SSLDisableOCSPEndpointCheck        bool
	SSLDisableOCSPEndpointCheckSet     bool
	Timeout                            time.Duration
	TimeoutSet                         bool
	WString                            string
	WNumber                            int
	WNumberSet                         bool
	Username                           string
	UsernameSet                        bool
	ZlibLevel                          int
	ZlibLevelSet                       bool
	ZstdLevel                          int
	ZstdLevelSet                       bool

	WTimeout              time.Duration
	WTimeoutSet           bool
	WTimeoutSetFromOption bool

	Options        map[string][]string
	UnknownOptions map[string][]string
}

func (u *ConnString) String() string {
	return u.Original
}

// HasAuthParameters returns true if this ConnString has any authentication parameters set and therefore represents
// a request for authentication.
func (u *ConnString) HasAuthParameters() bool {
	// Check all auth parameters except for AuthSource because an auth source without other credentials is semantically
	// valid and must not be interpreted as a request for authentication.
	return u.AuthMechanism != "" || u.AuthMechanismProperties != nil || u.UsernameSet || u.PasswordSet
}

// Validate checks that the Auth and SSL parameters are valid values.
func (u *ConnString) Validate() error {
	p := parser{
		dnsResolver: dns.DefaultResolver,
		ConnString:  *u,
	}
	return p.validate()
}

// ConnectMode informs the driver on how to connect
// to the server.
type ConnectMode uint8

var _ fmt.Stringer = ConnectMode(0)

// ConnectMode constants.
const (
	AutoConnect ConnectMode = iota
	SingleConnect
)

// String implements the fmt.Stringer interface.
func (c ConnectMode) String() string {
	switch c {
	case AutoConnect:
		return "automatic"
	case SingleConnect:
		return "direct"
	default:
		return "unknown"
	}
}

// Scheme constants
const (
	SchemeMongoDB    = "mongodb"
	SchemeMongoDBSRV = "mongodb+srv"
)

type parser struct {
	ConnString

	dnsResolver *dns.Resolver
	tlsssl      *bool // used to determine if tls and ssl options are both specified and set differently.
}

func (p *parser) parse(original string) error {
	p.Original = original
	uri := original

	var err error
	if strings.HasPrefix(uri, SchemeMongoDBSRV+"://") {
		p.Scheme = SchemeMongoDBSRV
		// remove the scheme
		uri = uri[len(SchemeMongoDBSRV)+3:]
	} else if strings.HasPrefix(uri, SchemeMongoDB+"://") {
		p.Scheme = SchemeMongoDB
		// remove the scheme
		uri = uri[len(SchemeMongoDB)+3:]
	} else {
		return errors.New(`scheme must be "mongodb" or "mongodb+srv"`)
	}

	if idx := strings.Index(uri, "@"); idx != -1 {
		userInfo := uri[:idx]
		uri = uri[idx+1:]

		username := userInfo
		var password string

		if idx := strings.Index(userInfo, ":"); idx != -1 {
			username = userInfo[:idx]
			password = userInfo[idx+1:]
			p.PasswordSet = true
		}

		// Validate and process the username.
		if strings.Contains(username, "/") {
			return fmt.Errorf("unescaped slash in username")
		}
		p.Username, err = url.PathUnescape(username)
		if err != nil {
			return fmt.Errorf("invalid username: %w", err)
		}
		p.UsernameSet = true

		// Validate and process the password.
		if strings.Contains(password, ":") {
			return fmt.Errorf("unescaped colon in password")
		}
		if strings.Contains(password, "/") {
			return fmt.Errorf("unescaped slash in password")
		}
		p.Password, err = url.PathUnescape(password)
		if err != nil {
			return fmt.Errorf("invalid password: %w", err)
		}
	}

	// fetch the hosts field
	hosts := uri
	if idx := strings.IndexAny(uri, "/?@"); idx != -1 {
		if uri[idx] == '@' {
			return fmt.Errorf("unescaped @ sign in user info")
		}
		if uri[idx] == '?' {
			return fmt.Errorf("must have a / before the query ?")
		}
		hosts = uri[:idx]
	}

	parsedHosts := strings.Split(hosts, ",")
	uri = uri[len(hosts):]
	extractedDatabase, err := extractDatabaseFromURI(uri)
	if err != nil {
		return err
	}

	uri = extractedDatabase.uri
	p.Database = extractedDatabase.db

	// grab connection arguments from URI
	connectionArgsFromQueryString, err := extractQueryArgsFromURI(uri)
	if err != nil {
		return err
	}

	// grab connection arguments from TXT record and enable SSL if "mongodb+srv://"
	var connectionArgsFromTXT []string
	if p.Scheme == SchemeMongoDBSRV {
		connectionArgsFromTXT, err = p.dnsResolver.GetConnectionArgsFromTXT(hosts)
		if err != nil {
			return err
		}

		// SSL is enabled by default for SRV, but can be manually disabled with "ssl=false".
		p.SSL = true
		p.SSLSet = true
	}

	// add connection arguments from URI and TXT records to connstring
	connectionArgPairs := make([]string, 0, len(connectionArgsFromTXT)+len(connectionArgsFromQueryString))
	connectionArgPairs = append(connectionArgPairs, connectionArgsFromTXT...)
	connectionArgPairs = append(connectionArgPairs, connectionArgsFromQueryString...)

	for _, pair := range connectionArgPairs {
		err := p.addOption(pair)
		if err != nil {
			return err
		}
	}

	// do SRV lookup if "mongodb+srv://"
	if p.Scheme == SchemeMongoDBSRV {
		parsedHosts, err = p.dnsResolver.ParseHosts(hosts, p.SRVServiceName, true)
		if err != nil {
			return err
		}

		// If p.SRVMaxHosts is non-zero and is less than the number of hosts, randomly
		// select SRVMaxHosts hosts from parsedHosts.
		if p.SRVMaxHosts > 0 && p.SRVMaxHosts < len(parsedHosts) {
			random.Shuffle(len(parsedHosts), func(i, j int) {
				parsedHosts[i], parsedHosts[j] = parsedHosts[j], parsedHosts[i]
			})
			parsedHosts = parsedHosts[:p.SRVMaxHosts]
		}
	}

	for _, host := range parsedHosts {
		err = p.addHost(host)
		if err != nil {
			return fmt.Errorf("invalid host %q: %w", host, err)
		}
	}
	if len(p.Hosts) == 0 {
		return fmt.Errorf("must have at least 1 host")
	}

	err = p.setDefaultAuthParams(extractedDatabase.db)
	if err != nil {
		return err
	}

	// If WTimeout was set from manual options passed in, set WTImeoutSet to true.
	if p.WTimeoutSetFromOption {
		p.WTimeoutSet = true
	}

	return nil
}

func (p *parser) validate() error {
	var err error

	err = p.validateAuth()
	if err != nil {
		return err
	}

	if err = p.validateSSL(); err != nil {
		return err
	}

	// Check for invalid write concern (i.e. w=0 and j=true)
	if p.WNumberSet && p.WNumber == 0 && p.JSet && p.J {
		return writeconcern.ErrInconsistent
	}

	// Check for invalid use of direct connections.
	if (p.ConnectSet && p.Connect == SingleConnect) || (p.DirectConnectionSet && p.DirectConnection) {
		if len(p.Hosts) > 1 {
			return errors.New("a direct connection cannot be made if multiple hosts are specified")
		}
		if p.Scheme == SchemeMongoDBSRV {
			return errors.New("a direct connection cannot be made if an SRV URI is used")
		}
		if p.LoadBalancedSet && p.LoadBalanced {
			return ErrLoadBalancedWithDirectConnection
		}
	}

	// Validation for load-balanced mode.
	if p.LoadBalancedSet && p.LoadBalanced {
		if len(p.Hosts) > 1 {
			return ErrLoadBalancedWithMultipleHosts
		}
		if p.ReplicaSet != "" {
			return ErrLoadBalancedWithReplicaSet
		}
	}

	// Check for invalid use of SRVMaxHosts.
	if p.SRVMaxHosts > 0 {
		if p.ReplicaSet != "" {
			return ErrSRVMaxHostsWithReplicaSet
		}
		if p.LoadBalanced {
			return ErrSRVMaxHostsWithLoadBalanced
		}
	}

	return nil
}

func (p *parser) setDefaultAuthParams(dbName string) error {
	// We do this check here rather than in validateAuth because this function is called as part of parsing and sets
	// the value of AuthSource if authentication is enabled.
	if p.AuthSourceSet && p.AuthSource == "" {
		return errors.New("authSource must be non-empty when supplied in a URI")
	}

	switch strings.ToLower(p.AuthMechanism) {
	case "plain":
		if p.AuthSource == "" {
			p.AuthSource = dbName
			if p.AuthSource == "" {
				p.AuthSource = "$external"
			}
		}
	case "gssapi":
		if p.AuthMechanismProperties == nil {
			p.AuthMechanismProperties = map[string]string{
				"SERVICE_NAME": "mongodb",
			}
		} else if v, ok := p.AuthMechanismProperties["SERVICE_NAME"]; !ok || v == "" {
			p.AuthMechanismProperties["SERVICE_NAME"] = "mongodb"
		}
		fallthrough
	case "mongodb-aws", "mongodb-x509":
		if p.AuthSource == "" {
			p.AuthSource = "$external"
		} else if p.AuthSource != "$external" {
			return fmt.Errorf("auth source must be $external")
		}
	case "mongodb-cr":
		fallthrough
	case "scram-sha-1":
		fallthrough
	case "scram-sha-256":
		if p.AuthSource == "" {
			p.AuthSource = dbName
			if p.AuthSource == "" {
				p.AuthSource = "admin"
			}
		}
	case "":
		// Only set auth source if there is a request for authentication via non-empty credentials.
		if p.AuthSource == "" && (p.AuthMechanismProperties != nil || p.Username != "" || p.PasswordSet) {
			p.AuthSource = dbName
			if p.AuthSource == "" {
				p.AuthSource = "admin"
			}
		}
	default:
		return fmt.Errorf("invalid auth mechanism")
	}
	return nil
}

func (p *parser) validateAuth() error {
	switch strings.ToLower(p.AuthMechanism) {
	case "mongodb-cr":
		if p.Username == "" {
			return fmt.Errorf("username required for MONGO-CR")
		}
		if p.Password == "" {
			return fmt.Errorf("password required for MONGO-CR")
		}
		if p.AuthMechanismProperties != nil {
			return fmt.Errorf("MONGO-CR cannot have mechanism properties")
		}
	case "mongodb-x509":
		if p.Password != "" {
			return fmt.Errorf("password cannot be specified for MONGO-X509")
		}
		if p.AuthMechanismProperties != nil {
			return fmt.Errorf("MONGO-X509 cannot have mechanism properties")
		}
	case "mongodb-aws":
		if p.Username != "" && p.Password == "" {
			return fmt.Errorf("username without password is invalid for MONGODB-AWS")
		}
		if p.Username == "" && p.Password != "" {
			return fmt.Errorf("password without username is invalid for MONGODB-AWS")
		}
		var token bool
		for k := range p.AuthMechanismProperties {
			if k != "AWS_SESSION_TOKEN" {
				return fmt.Errorf("invalid auth property for MONGODB-AWS")
			}
			token = true
		}
		if token && p.Username == "" && p.Password == "" {
			return fmt.Errorf("token without username and password is invalid for MONGODB-AWS")
		}
	case "gssapi":
		if p.Username == "" {
			return fmt.Errorf("username required for GSSAPI")
		}
		for k := range p.AuthMechanismProperties {
			if k != "SERVICE_NAME" && k != "CANONICALIZE_HOST_NAME" && k != "SERVICE_REALM" && k != "SERVICE_HOST" {
				return fmt.Errorf("invalid auth property for GSSAPI")
			}
		}
	case "plain":
		if p.Username == "" {
			return fmt.Errorf("username required for PLAIN")
		}
		if p.Password == "" {
			return fmt.Errorf("password required for PLAIN")
		}
		if p.AuthMechanismProperties != nil {
			return fmt.Errorf("PLAIN cannot have mechanism properties")
		}
	case "scram-sha-1":
		if p.Username == "" {
			return fmt.Errorf("username required for SCRAM-SHA-1")
		}
		if p.Password == "" {
			return fmt.Errorf("password required for SCRAM-SHA-1")
		}
		if p.AuthMechanismProperties != nil {
			return fmt.Errorf("SCRAM-SHA-1 cannot have mechanism properties")
		}
	case "scram-sha-256":
		if p.Username == "" {
			return fmt.Errorf("username required for SCRAM-SHA-256")
		}
		if p.Password == "" {
			return fmt.Errorf("password required for SCRAM-SHA-256")
		}
		if p.AuthMechanismProperties != nil {
			return fmt.Errorf("SCRAM-SHA-256 cannot have mechanism properties")
		}
	case "":
		if p.UsernameSet && p.Username == "" {
			return fmt.Errorf("username required if URI contains user info")
		}
	default:
		return fmt.Errorf("invalid auth mechanism")
	}
	return nil
}

func (p *parser) validateSSL() error {
	if !p.SSL {
		return nil
	}

	if p.SSLClientCertificateKeyFileSet {
		if p.SSLCertificateFileSet || p.SSLPrivateKeyFileSet {
			return errors.New("the sslClientCertificateKeyFile/tlsCertificateKeyFile URI option cannot be provided " +
				"along with tlsCertificateFile or tlsPrivateKeyFile")
		}
		return nil
	}
	if p.SSLCertificateFileSet && !p.SSLPrivateKeyFileSet {
		return errors.New("the tlsPrivateKeyFile URI option must be provided if the tlsCertificateFile option is specified")
	}
	if p.SSLPrivateKeyFileSet && !p.SSLCertificateFileSet {
		return errors.New("the tlsCertificateFile URI option must be provided if the tlsPrivateKeyFile option is specified")
	}

	if p.SSLInsecureSet && p.SSLDisableOCSPEndpointCheckSet {
		return errors.New("the sslInsecure/tlsInsecure URI option cannot be provided along with " +
			"tlsDisableOCSPEndpointCheck ")
	}
	return nil
}

func (p *parser) addHost(host string) error {
	if host == "" {
		return nil
	}
	host, err := url.QueryUnescape(host)
	if err != nil {
		return fmt.Errorf("invalid host %q: %w", host, err)
	}

	_, port, err := net.SplitHostPort(host)
	// this is unfortunate that SplitHostPort actually requires
	// a port to exist.
	if err != nil {
		if addrError, ok := err.(*net.AddrError); !ok || addrError.Err != "missing port in address" {
			return err
		}
	}

	if port != "" {
		d, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("port must be an integer: %w", err)
		}
		if d <= 0 || d >= 65536 {
			return fmt.Errorf("port must be in the range [1, 65535]")
		}
	}
	p.Hosts = append(p.Hosts, host)
	return nil
}

// IsValidServerMonitoringMode will return true if the given string matches a
// valid server monitoring mode.
func IsValidServerMonitoringMode(mode string) bool {
	return mode == ServerMonitoringModeAuto ||
		mode == ServerMonitoringModeStream ||
		mode == ServerMonitoringModePoll
}

func (p *parser) addOption(pair string) error {
	kv := strings.SplitN(pair, "=", 2)
	if len(kv) != 2 || kv[0] == "" {
		return fmt.Errorf("invalid option")
	}

	key, err := url.QueryUnescape(kv[0])
	if err != nil {
		return fmt.Errorf("invalid option key %q: %w", kv[0], err)
	}

	value, err := url.QueryUnescape(kv[1])
	if err != nil {
		return fmt.Errorf("invalid option value %q: %w", kv[1], err)
	}

	lowerKey := strings.ToLower(key)
	switch lowerKey {
	case "appname":
		p.AppName = value
	case "authmechanism":
		p.AuthMechanism = value
	case "authmechanismproperties":
		p.AuthMechanismProperties = make(map[string]string)
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) != 2 || kv[0] == "" {
				return fmt.Errorf("invalid authMechanism property")
			}
			p.AuthMechanismProperties[kv[0]] = kv[1]
		}
		p.AuthMechanismPropertiesSet = true
	case "authsource":
		p.AuthSource = value
		p.AuthSourceSet = true
	case "compressors":
		compressors := strings.Split(value, ",")
		if len(compressors) < 1 {
			return fmt.Errorf("must have at least 1 compressor")
		}
		p.Compressors = compressors
	case "connect":
		switch strings.ToLower(value) {
		case "automatic":
		case "direct":
			p.Connect = SingleConnect
		default:
			return fmt.Errorf("invalid 'connect' value: %q", value)
		}
		if p.DirectConnectionSet {
			expectedValue := p.Connect == SingleConnect // directConnection should be true if connect=direct
			if p.DirectConnection != expectedValue {
				return fmt.Errorf("options connect=%q and directConnection=%v conflict", value, p.DirectConnection)
			}
		}

		p.ConnectSet = true
	case "directconnection":
		switch strings.ToLower(value) {
		case "true":
			p.DirectConnection = true
		case "false":
		default:
			return fmt.Errorf("invalid 'directConnection' value: %q", value)
		}

		if p.ConnectSet {
			expectedValue := AutoConnect
			if p.DirectConnection {
				expectedValue = SingleConnect
			}

			if p.Connect != expectedValue {
				return fmt.Errorf("options connect=%q and directConnection=%q conflict", p.Connect, value)
			}
		}
		p.DirectConnectionSet = true
	case "connecttimeoutms":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.ConnectTimeout = time.Duration(n) * time.Millisecond
		p.ConnectTimeoutSet = true
	case "heartbeatintervalms", "heartbeatfrequencyms":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.HeartbeatInterval = time.Duration(n) * time.Millisecond
		p.HeartbeatIntervalSet = true
	case "journal":
		switch value {
		case "true":
			p.J = true
		case "false":
			p.J = false
		default:
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		p.JSet = true
	case "loadbalanced":
		switch value {
		case "true":
			p.LoadBalanced = true
		case "false":
			p.LoadBalanced = false
		default:
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		p.LoadBalancedSet = true
	case "localthresholdms":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.LocalThreshold = time.Duration(n) * time.Millisecond
		p.LocalThresholdSet = true
	case "maxidletimems":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.MaxConnIdleTime = time.Duration(n) * time.Millisecond
		p.MaxConnIdleTimeSet = true
	case "maxpoolsize":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.MaxPoolSize = uint64(n)
		p.MaxPoolSizeSet = true
	case "minpoolsize":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.MinPoolSize = uint64(n)
		p.MinPoolSizeSet = true
	case "maxconnecting":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.MaxConnecting = uint64(n)
		p.MaxConnectingSet = true
	case "readconcernlevel":
		p.ReadConcernLevel = value
	case "readpreference":
		p.ReadPreference = value
	case "readpreferencetags":
		if value == "" {
			// If "readPreferenceTags=" is supplied, append an empty map to tag sets to
			// represent a wild-card.
			p.ReadPreferenceTagSets = append(p.ReadPreferenceTagSets, map[string]string{})
			break
		}

		tags := make(map[string]string)
		items := strings.Split(value, ",")
		for _, item := range items {
			parts := strings.Split(item, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid value for %q: %q", key, value)
			}
			tags[parts[0]] = parts[1]
		}
		p.ReadPreferenceTagSets = append(p.ReadPreferenceTagSets, tags)
	case "maxstaleness", "maxstalenessseconds":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.MaxStaleness = time.Duration(n) * time.Second
		p.MaxStalenessSet = true
	case "replicaset":
		p.ReplicaSet = value
	case "retrywrites":
		switch value {
		case "true":
			p.RetryWrites = true
		case "false":
			p.RetryWrites = false
		default:
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		p.RetryWritesSet = true
	case "retryreads":
		switch value {
		case "true":
			p.RetryReads = true
		case "false":
			p.RetryReads = false
		default:
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		p.RetryReadsSet = true
	case "servermonitoringmode":
		if !IsValidServerMonitoringMode(value) {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		p.ServerMonitoringMode = value
	case "serverselectiontimeoutms":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.ServerSelectionTimeout = time.Duration(n) * time.Millisecond
		p.ServerSelectionTimeoutSet = true
	case "sockettimeoutms":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.SocketTimeout = time.Duration(n) * time.Millisecond
		p.SocketTimeoutSet = true
	case "srvmaxhosts":
		// srvMaxHosts can only be set on URIs with the "mongodb+srv" scheme
		if p.Scheme != SchemeMongoDBSRV {
			return fmt.Errorf("cannot specify srvMaxHosts on non-SRV URI")
		}

		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.SRVMaxHosts = n
	case "srvservicename":
		// srvServiceName can only be set on URIs with the "mongodb+srv" scheme
		if p.Scheme != SchemeMongoDBSRV {
			return fmt.Errorf("cannot specify srvServiceName on non-SRV URI")
		}

		// srvServiceName must be between 1 and 62 characters according to
		// our specification. Empty service names are not valid, and the service
		// name (including prepended underscore) should not exceed the 63 character
		// limit for DNS query subdomains.
		if len(value) < 1 || len(value) > 62 {
			return fmt.Errorf("srvServiceName value must be between 1 and 62 characters")
		}
		p.SRVServiceName = value
	case "ssl", "tls":
		switch value {
		case "true":
			p.SSL = true
		case "false":
			p.SSL = false
		default:
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		if p.tlsssl != nil && *p.tlsssl != p.SSL {
			return errors.New("tls and ssl options, when both specified, must be equivalent")
		}

		p.tlsssl = new(bool)
		*p.tlsssl = p.SSL

		p.SSLSet = true
	case "sslclientcertificatekeyfile", "tlscertificatekeyfile":
		p.SSL = true
		p.SSLSet = true
		p.SSLClientCertificateKeyFile = value
		p.SSLClientCertificateKeyFileSet = true
	case "sslclientcertificatekeypassword", "tlscertificatekeyfilepassword":
		p.SSLClientCertificateKeyPassword = func() string { return value }
		p.SSLClientCertificateKeyPasswordSet = true
	case "tlscertificatefile":
		p.SSL = true
		p.SSLSet = true
		p.SSLCertificateFile = value
		p.SSLCertificateFileSet = true
	case "tlsprivatekeyfile":
		p.SSL = true
		p.SSLSet = true
		p.SSLPrivateKeyFile = value
		p.SSLPrivateKeyFileSet = true
	case "sslinsecure", "tlsinsecure":
		switch value {
		case "true":
			p.SSLInsecure = true
		case "false":
			p.SSLInsecure = false
		default:
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		p.SSLInsecureSet = true
	case "sslcertificateauthorityfile", "tlscafile":
		p.SSL = true
		p.SSLSet = true
		p.SSLCaFile = value
		p.SSLCaFileSet = true
	case "timeoutms":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.Timeout = time.Duration(n) * time.Millisecond
		p.TimeoutSet = true
	case "tlsdisableocspendpointcheck":
		p.SSL = true
		p.SSLSet = true

		switch value {
		case "true":
			p.SSLDisableOCSPEndpointCheck = true
		case "false":
			p.SSLDisableOCSPEndpointCheck = false
		default:
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.SSLDisableOCSPEndpointCheckSet = true
	case "w":
		if w, err := strconv.Atoi(value); err == nil {
			if w < 0 {
				return fmt.Errorf("invalid value for %q: %q", key, value)
			}

			p.WNumber = w
			p.WNumberSet = true
			p.WString = ""
			break
		}

		p.WString = value
		p.WNumberSet = false

	case "wtimeoutms":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.WTimeout = time.Duration(n) * time.Millisecond
		p.WTimeoutSet = true
	case "wtimeout":
		// Defer to wtimeoutms, but not to a manually-set option.
		if p.WTimeoutSet {
			break
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}
		p.WTimeout = time.Duration(n) * time.Millisecond
	case "zlibcompressionlevel":
		level, err := strconv.Atoi(value)
		if err != nil || (level < -1 || level > 9) {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		if level == -1 {
			level = wiremessage.DefaultZlibLevel
		}
		p.ZlibLevel = level
		p.ZlibLevelSet = true
	case "zstdcompressionlevel":
		const maxZstdLevel = 22 // https://github.com/facebook/zstd/blob/a880ca239b447968493dd2fed3850e766d6305cc/contrib/linux-kernel/lib/zstd/compress.c#L3291
		level, err := strconv.Atoi(value)
		if err != nil || (level < -1 || level > maxZstdLevel) {
			return fmt.Errorf("invalid value for %q: %q", key, value)
		}

		if level == -1 {
			level = wiremessage.DefaultZstdLevel
		}
		p.ZstdLevel = level
		p.ZstdLevelSet = true
	default:
		if p.UnknownOptions == nil {
			p.UnknownOptions = make(map[string][]string)
		}
		p.UnknownOptions[lowerKey] = append(p.UnknownOptions[lowerKey], value)
	}

	if p.Options == nil {
		p.Options = make(map[string][]string)
	}
	p.Options[lowerKey] = append(p.Options[lowerKey], value)

	return nil
}

func extractQueryArgsFromURI(uri string) ([]string, error) {
	if len(uri) == 0 {
		return nil, nil
	}

	if uri[0] != '?' {
		return nil, errors.New("must have a ? separator between path and query")
	}

	uri = uri[1:]
	if len(uri) == 0 {
		return nil, nil
	}
	return strings.FieldsFunc(uri, func(r rune) bool { return r == ';' || r == '&' }), nil

}

type extractedDatabase struct {
	uri string
	db  string
}

// extractDatabaseFromURI is a helper function to retrieve information about
// the database from the passed in URI. It accepts as an argument the currently
// parsed URI and returns the remainder of the uri, the database it found,
// and any error it encounters while parsing.
func extractDatabaseFromURI(uri string) (extractedDatabase, error) {
	if len(uri) == 0 {
		return extractedDatabase{}, nil
	}

	if uri[0] != '/' {
		return extractedDatabase{}, errors.New("must have a / separator between hosts and path")
	}

	uri = uri[1:]
	if len(uri) == 0 {
		return extractedDatabase{}, nil
	}

	database := uri
	if idx := strings.IndexRune(uri, '?'); idx != -1 {
		database = uri[:idx]
	}

	escapedDatabase, err := url.QueryUnescape(database)
	if err != nil {
		return extractedDatabase{}, fmt.Errorf("invalid database %q: %w", database, err)
	}

	uri = uri[len(database):]

	return extractedDatabase{
		uri: uri,
		db:  escapedDatabase,
	}, nil
}
