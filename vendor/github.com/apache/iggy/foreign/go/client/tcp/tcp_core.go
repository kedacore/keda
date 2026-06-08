// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tcp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
	ierror "github.com/apache/iggy/foreign/go/errors"
	"github.com/apache/iggy/foreign/go/internal/command"
	"github.com/avast/retry-go/v5"
)

type Option func(config *Options)

type Options struct {
	config config
}

func GetDefaultOptions() Options {
	return Options{
		config: defaultTcpClientConfig(),
	}
}

type IggyTcpClient struct {
	conn                   net.Conn
	mtx                    sync.Mutex
	config                 config
	MessageCompression     iggcon.IggyMessageCompression
	leaderRedirectionState iggcon.LeaderRedirectionState
	clientAddress          string
	currentServerAddress   string
	connectedAt            time.Time
	state                  iggcon.State
}

type config struct {
	// serverAddress is the address of the Iggy server
	serverAddress string
	// tlsEnabled indicates whether to use TLS when connecting to the server
	tlsEnabled bool
	tls        tlsConfig
	// autoLogin indicates whether to automatically login user after establishing connection.
	autoLogin AutoLogin
	// reconnection indicates whether to automatically reconnect when disconnected
	reconnection tcpClientReconnectionConfig
	// noDelay disable Nagle's algorithm for the TCP connection
	noDelay bool
}

func defaultTcpClientConfig() config {
	return config{
		serverAddress: "127.0.0.1:8090",
		tlsEnabled:    false,
		tls:           defaultTLSConfig(),
		autoLogin:     AutoLogin{},
		reconnection:  defaultTcpClientReconnectionConfig(),
		noDelay:       false,
	}
}

type tcpClientReconnectionConfig struct {
	enabled          bool
	maxRetries       uint32
	interval         time.Duration
	reestablishAfter time.Duration
}

func defaultTcpClientReconnectionConfig() tcpClientReconnectionConfig {
	return tcpClientReconnectionConfig{
		enabled:          true,
		maxRetries:       0, //infinity retry
		interval:         2 * time.Second,
		reestablishAfter: 0,
	}
}

type tlsConfig struct {
	// tlsDomain is the domain to use for TLS when connecting to the server
	// If empty, automatically extracts the hostname/IP from serverAddress
	tlsDomain string
	// tlsCAFile is the path to the CA file to use for TLS
	tlsCAFile string
	// tlsValidateCertificate indicates whether to validate the server's TLS certificate
	tlsValidateCertificate bool
}

func defaultTLSConfig() tlsConfig {
	return tlsConfig{
		tlsDomain:              "",
		tlsCAFile:              "",
		tlsValidateCertificate: true,
	}
}

type AutoLogin struct {
	enabled     bool
	credentials Credentials
}

func NewAutoLogin(credentials Credentials) AutoLogin {
	return AutoLogin{
		enabled:     true,
		credentials: credentials,
	}
}

type Credentials struct {
	username            string
	password            string
	personalAccessToken string
}

func NewUsernamePasswordCredentials(username, password string) Credentials {
	return Credentials{
		username: username,
		password: password,
	}
}

func NewPersonalAccessTokenCredentials(token string) Credentials {
	return Credentials{
		personalAccessToken: token,
	}
}

// WithServerAddress Sets the server address for the TCP client.
func WithServerAddress(address string) Option {
	return func(opts *Options) {
		opts.config.serverAddress = address
	}
}

// TLSOption is a functional option for configuring TLS settings.
type TLSOption func(cfg *tlsConfig)

// WithTLS enables TLS for the TCP client and applies the given TLS options.
func WithTLS(tlsOpts ...TLSOption) Option {
	return func(opts *Options) {
		opts.config.tlsEnabled = true
		for _, tlsOpt := range tlsOpts {
			if tlsOpt != nil {
				tlsOpt(&opts.config.tls)
			}
		}
	}
}

// WithTLSDomain sets the TLS domain for server name indication (SNI).
// If not provided, the domain will be automatically extracted from the server address.
func WithTLSDomain(domain string) TLSOption {
	return func(cfg *tlsConfig) {
		cfg.tlsDomain = domain
	}
}

// WithTLSCAFile sets the path to the CA certificate file for TLS verification.
func WithTLSCAFile(path string) TLSOption {
	return func(cfg *tlsConfig) {
		cfg.tlsCAFile = path
	}
}

// WithTLSValidateCertificate enables or disables TLS certificate validation.
func WithTLSValidateCertificate(validate bool) TLSOption {
	return func(cfg *tlsConfig) {
		cfg.tlsValidateCertificate = validate
	}
}

// NewIggyTcpClient creates a new Iggy TCP client with the given options.
// warning: don't use this function directly, use iggycli.NewIggyClient with iggycli.WithTcp instead.
func NewIggyTcpClient(options ...Option) (*IggyTcpClient, error) {
	opts := GetDefaultOptions()
	for _, opt := range options {
		if opt != nil {
			opt(&opts)
		}
	}

	cli := &IggyTcpClient{
		config:                 opts.config,
		clientAddress:          "",
		conn:                   nil,
		state:                  iggcon.StateDisconnected,
		connectedAt:            time.Time{},
		leaderRedirectionState: iggcon.LeaderRedirectionState{},
		currentServerAddress:   opts.config.serverAddress,
	}

	if err := cli.connect(); err != nil {
		return nil, err
	}

	return cli, nil
}

const (
	RequestInitialBytesLength  = 4
	ResponseInitialBytesLength = 8
	MaxStringLength            = 255
	MaxPartitionCount          = 1000
)

func (c *IggyTcpClient) read(expectedSize int) (int, []byte, error) {
	var totalRead int
	buffer := make([]byte, expectedSize)

	for totalRead < expectedSize {
		readSize := expectedSize - totalRead
		n, err := c.conn.Read(buffer[totalRead : totalRead+readSize])
		if err != nil {
			return totalRead, buffer[:totalRead], err
		}
		totalRead += n
	}

	return totalRead, buffer, nil
}

func (c *IggyTcpClient) write(payload []byte) (int, error) {
	var totalWritten int
	for totalWritten < len(payload) {
		n, err := c.conn.Write(payload[totalWritten:])
		if err != nil {
			return totalWritten, err
		}
		totalWritten += n
	}

	return totalWritten, nil
}

// do sends the command to the Iggy server and returns the response.
func (c *IggyTcpClient) do(cmd command.Command) ([]byte, error) {
	data, err := cmd.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return c.sendAndFetchResponse(data, cmd.Code())
}

func (c *IggyTcpClient) sendAndFetchResponse(message []byte, command command.Code) ([]byte, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	payload := createPayload(message, command)
	if _, err := c.write(payload); err != nil {
		return nil, err
	}

	readBytes, buffer, err := c.read(ResponseInitialBytesLength)
	if err != nil {
		return nil, fmt.Errorf("failed to read response for TCP request: %w", err)
	}

	if readBytes != ResponseInitialBytesLength {
		return nil, fmt.Errorf("received an invalid or empty response: %w", ierror.EmptyResponse{})
	}

	if status := ierror.Code(binary.LittleEndian.Uint32(buffer[0:4])); status != 0 {
		return nil, ierror.FromCode(status)
	}

	length := int(binary.LittleEndian.Uint32(buffer[4:]))
	if length <= 1 {
		return []byte{}, nil
	}

	_, buffer, err = c.read(length)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

func createPayload(message []byte, command command.Code) []byte {
	messageLength := len(message) + 4
	messageBytes := make([]byte, RequestInitialBytesLength+messageLength)
	binary.LittleEndian.PutUint32(messageBytes[:4], uint32(messageLength))
	binary.LittleEndian.PutUint32(messageBytes[4:8], uint32(command))
	copy(messageBytes[8:], message)
	return messageBytes
}

func (c *IggyTcpClient) GetConnectionInfo() *iggcon.ConnectionInfo {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return &iggcon.ConnectionInfo{
		Protocol:      iggcon.Tcp,
		ServerAddress: c.currentServerAddress,
	}
}

func (c *IggyTcpClient) connect() error {
	c.mtx.Lock()
	switch c.state {
	case iggcon.StateShutdown:
		c.mtx.Unlock()
		return ierror.ErrClientShutdown
	case iggcon.StateConnected,
		iggcon.StateAuthenticating,
		iggcon.StateAuthenticated,
		iggcon.StateConnecting:
		c.mtx.Unlock()
		return nil
	default:
		c.state = iggcon.StateConnecting
	}
	connectedAt := c.connectedAt
	c.mtx.Unlock()

	// handle reestablish interval
	if !connectedAt.IsZero() {
		now := time.Now()
		elapsed := now.Sub(connectedAt)
		interval := c.config.reconnection.reestablishAfter

		if elapsed < interval {
			remaining := interval - elapsed
			time.Sleep(remaining)
		}
	}
	attempts := uint(1)
	interval := time.Duration(0)
	if c.config.reconnection.enabled {
		attempts = uint(c.config.reconnection.maxRetries)
		interval = c.config.reconnection.interval
	}

	var conn net.Conn
	if err := retry.New(
		retry.Attempts(attempts),
		retry.Delay(interval),
		retry.DelayType(retry.FixedDelay),
	).Do(
		func() error {
			connection, err := net.Dial("tcp", c.currentServerAddress)
			if err != nil {
				return ierror.ErrCannotEstablishConnection
			}

			tc := connection.(*net.TCPConn)
			_ = tc.SetNoDelay(c.config.noDelay)

			c.mtx.Lock()
			c.clientAddress = tc.LocalAddr().String()
			c.mtx.Unlock()

			if !c.config.tlsEnabled {
				conn = connection
				return nil
			}

			// TLS logic
			tlsConfig, err := c.createTLSConfig()
			if err != nil {
				_ = connection.Close()
				return err
			}

			tlsConn := tls.Client(connection, tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				_ = connection.Close()
				return fmt.Errorf("TLS handshake failed: %w", err)
			}

			conn = tlsConn
			return nil
		}); err != nil {
		c.mtx.Lock()
		c.state = iggcon.StateDisconnected
		c.mtx.Unlock()
		// TODO publish event disconnected
		return err
	}

	c.mtx.Lock()
	c.conn = conn
	c.state = iggcon.StateConnected
	c.connectedAt = time.Now()
	c.mtx.Unlock()
	return nil
}

func (c *IggyTcpClient) createTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !c.config.tls.tlsValidateCertificate,
	}

	// Set server name for SNI
	serverName := c.config.tls.tlsDomain
	if serverName == "" {
		// Extract hostname from server address (format: "host:port")
		host := c.currentServerAddress
		if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
			host = host[:colonIdx]
		}
		serverName = host
	}

	if serverName == "" {
		return nil, ierror.ErrInvalidTlsDomain
	}
	tlsConfig.ServerName = serverName

	// Load CA certificate if provided
	if c.config.tls.tlsCAFile != "" {
		caCert, err := os.ReadFile(c.config.tls.tlsCAFile)
		if err != nil {
			return nil, ierror.ErrInvalidTlsCertificatePath
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, ierror.ErrInvalidTlsCertificate
		}

		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

func (c *IggyTcpClient) disconnect() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.state == iggcon.StateDisconnected {
		return nil
	}
	c.state = iggcon.StateDisconnected
	if err := c.conn.Close(); err != nil {
		return err
	}
	// TODO event pushing logic
	return nil
}

func (c *IggyTcpClient) shutdown() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.state == iggcon.StateShutdown {
		return nil
	}
	if err := c.conn.Close(); err != nil {
		return err
	}
	c.state = iggcon.StateShutdown
	// TODO push shutdown event
	return nil
}

func (c *IggyTcpClient) Close() error {
	return c.shutdown()
}
