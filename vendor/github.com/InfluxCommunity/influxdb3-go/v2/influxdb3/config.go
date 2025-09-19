/*
 The MIT License

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package influxdb3

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/influxdata/line-protocol/v2/lineprotocol"
)

const (
	envInfluxHost          = "INFLUX_HOST"
	envInfluxToken         = "INFLUX_TOKEN"
	envInfluxAuthScheme    = "INFLUX_AUTH_SCHEME"
	envInfluxOrg           = "INFLUX_ORG"
	envInfluxDatabase      = "INFLUX_DATABASE"
	envInfluxPrecision     = "INFLUX_PRECISION"
	envInfluxGzipThreshold = "INFLUX_GZIP_THRESHOLD"
	envInfluxWriteNoSync   = "INFLUX_WRITE_NO_SYNC"
)

const (
	connStrInfluxToken         = "token"
	connStrInfluxAuthScheme    = "authScheme"
	connStrInfluxOrg           = "org"
	connStrInfluxDatabase      = "database"
	connStrInfluxPrecision     = "precision"
	connStrInfluxGzipThreshold = "gzipThreshold"
	connStrInfluxWriteNoSync   = "writeNoSync"
)

const (
	// defaultTimeout specifies the default value of ClientConfig.Timeout.
	defaultTimeout = 10 * time.Second
	// defaultIdleConnectionTimeout specifies the default value of ClientConfig.IdleConnectionTimeout.
	defaultIdleConnectionTimeout = 90 * time.Second
	// defaultMaxIdleConnections specifies the default value of ClientConfig.MaxIdleConnections.
	defaultMaxIdleConnections = 100
)

// ClientConfig holds the parameters for creating a new client.
// The only mandatory field is Host. Token is also important
// if authentication was not done outside this client.
type ClientConfig struct {
	// Host holds the URL of the InfluxDB server to connect to.
	// This must be non-empty. E.g. http://localhost:8086
	Host string

	// Token holds the authorization token for the API.
	// This can be obtained through the GUI web browser interface.
	Token string

	// AuthScheme defines token authentication scheme. For example, "Token", "Bearer" etc.
	// Leave empty for InfluxDB Cloud access. Set to "Bearer" for InfluxDB Edge (OSS).
	AuthScheme string

	// Organization is name or ID of organization where data (databases, users, tasks, etc.) belongs to.
	// Optional for InfluxDB Cloud.
	Organization string

	// Database used by the client.
	Database string

	// HTTPClient is used to make API requests.
	//
	// This can be used to specify a custom TLS configuration
	// (TLSClientConfig), a custom request timeout (Timeout),
	// or other customization as required.
	//
	// If HTTPClient is nil, http.DefaultClient will be used with the following adjustments:
	//   - Timeout: 10s
	//   - Transport.IdleConnTimeout: 90s
	//   - Transport.MaxIdleConns: 100
	//   - Transport.MaxIdleConnsPerHost: 100
	HTTPClient *http.Client

	// Timeout specifies the overall time limit for requests made by the Client.
	// The timeout includes connection time, any redirects, and reading the response body.
	// It is applied to write (HTTP client) operations only.
	//
	// A negative value means no timeout. Default value: 10 seconds.
	Timeout time.Duration

	// IdleConnectionTimeout specifies the maximum amount of time an idle connection
	// will remain idle before closing itself.
	// It is applied to write (HTTP client) operations only.
	//
	// A negative value means no timeout. Default value: 90 seconds.
	IdleConnectionTimeout time.Duration

	// MaxIdleConnections controls the maximum number of idle connections.
	// It is applied to write (HTTP client) operations only.
	//
	// A negative value means no limit. Default value: 100.
	MaxIdleConnections int

	// Write options
	WriteOptions *WriteOptions

	// Default HTTP headers to be included in requests
	Headers http.Header

	// SSL root certificates file path
	SSLRootsFilePath string

	// Proxy URL
	Proxy string
}

// validate validates the config.
func (c *ClientConfig) validate() error {
	if c.Host == "" {
		return errors.New("empty host")
	}
	if c.Token == "" {
		return errors.New("no token specified")
	}

	return nil
}

// parse initializes the client config from provided connection string.
func (c *ClientConfig) parse(connectionString string) error {
	u, err := url.Parse(connectionString)
	if err != nil {
		return err
	}

	if !(u.Scheme == "http" || u.Scheme == "https") {
		return errors.New("only http or https is supported")
	}

	values := u.Query()

	u.RawQuery = ""
	c.Host = u.String()

	if token, ok := values[connStrInfluxToken]; ok {
		c.Token = token[0]
	}
	if authScheme, ok := values[connStrInfluxAuthScheme]; ok {
		c.AuthScheme = authScheme[0]
	}
	if org, ok := values[connStrInfluxOrg]; ok {
		c.Organization = org[0]
	}
	if database, ok := values[connStrInfluxDatabase]; ok {
		c.Database = database[0]
	}
	if precision, ok := values[connStrInfluxPrecision]; ok {
		if err := c.parsePrecision(precision[0]); err != nil {
			return err
		}
	}
	if gzipThreshold, ok := values[connStrInfluxGzipThreshold]; ok {
		if err := c.parseGzipThreshold(gzipThreshold[0]); err != nil {
			return err
		}
	}
	if writeNoSync, ok := values[connStrInfluxWriteNoSync]; ok {
		if err := c.parseWriteNoSync(writeNoSync[0]); err != nil {
			return err
		}
	}

	return nil
}

// env initializes the client config from environment variables.
func (c *ClientConfig) env() error {
	if host, ok := os.LookupEnv(envInfluxHost); ok {
		c.Host = host
	}
	if token, ok := os.LookupEnv(envInfluxToken); ok {
		c.Token = token
	}
	if authScheme, ok := os.LookupEnv(envInfluxAuthScheme); ok {
		c.AuthScheme = authScheme
	}
	if org, ok := os.LookupEnv(envInfluxOrg); ok {
		c.Organization = org
	}
	if database, ok := os.LookupEnv(envInfluxDatabase); ok {
		c.Database = database
	}
	if precision, ok := os.LookupEnv(envInfluxPrecision); ok {
		if err := c.parsePrecision(precision); err != nil {
			return err
		}
	}
	if gzipThreshold, ok := os.LookupEnv(envInfluxGzipThreshold); ok {
		if err := c.parseGzipThreshold(gzipThreshold); err != nil {
			return err
		}
	}
	if writeNoSync, ok := os.LookupEnv(envInfluxWriteNoSync); ok {
		if err := c.parseWriteNoSync(writeNoSync); err != nil {
			return err
		}
	}

	return nil
}

// parsePrecision parses and sets precision
func (c *ClientConfig) parsePrecision(precision string) error {
	if c.WriteOptions == nil {
		options := DefaultWriteOptions
		c.WriteOptions = &options
	}

	switch precision {
	case "ns", "nanosecond":
		c.WriteOptions.Precision = lineprotocol.Nanosecond
	case "us", "microsecond":
		c.WriteOptions.Precision = lineprotocol.Microsecond
	case "ms", "millisecond":
		c.WriteOptions.Precision = lineprotocol.Millisecond
	case "s", "second":
		c.WriteOptions.Precision = lineprotocol.Second
	default:
		return fmt.Errorf("unsupported precision '%s'", precision)
	}

	return nil
}

// parseGzipThreshold parses and sets gzip threshold
func (c *ClientConfig) parseGzipThreshold(threshold string) error {
	if c.WriteOptions == nil {
		options := DefaultWriteOptions
		c.WriteOptions = &options
	}

	value, err := strconv.Atoi(threshold)
	if err != nil {
		return err
	}

	c.WriteOptions.GzipThreshold = value

	return nil
}

// parseWriteNoSync parses and sets write option NoSync
func (c *ClientConfig) parseWriteNoSync(strVal string) error {
	if c.WriteOptions == nil {
		options := DefaultWriteOptions
		c.WriteOptions = &options
	}

	value, err := strconv.ParseBool(strVal)
	if err != nil {
		return err
	}

	c.WriteOptions.NoSync = value

	return nil
}

// isTimeoutSet returns whether the Timeout was set.
func (c *ClientConfig) isTimeoutSet() bool {
	return c.Timeout != 0
}

// getTimeoutOrDefault returns the Timeout or the default value if not set.
func (c *ClientConfig) getTimeoutOrDefault() time.Duration {
	if c.Timeout == 0 {
		// Not set, use the default.
		return defaultTimeout
	}
	if c.Timeout < 0 {
		// No timeout.
		return 0
	}
	return c.Timeout
}

// isIdleConnectionTimeoutSet returns whether the IdleConnectionTimeout was set.
func (c *ClientConfig) isIdleConnectionTimeoutSet() bool {
	return c.IdleConnectionTimeout != 0
}

// getIdleConnectionTimeoutOrDefault returns the IdleConnectionTimeout or the default value if not set.
func (c *ClientConfig) getIdleConnectionTimeoutOrDefault() time.Duration {
	if c.IdleConnectionTimeout == 0 {
		// Not set, use the default.
		return defaultIdleConnectionTimeout
	}
	if c.IdleConnectionTimeout < 0 {
		// No timeout.
		return 0
	}
	return c.IdleConnectionTimeout
}

// isMaxIdleConnectionsSet returns whether the MaxIdleConnections was set.
func (c *ClientConfig) isMaxIdleConnectionsSet() bool {
	return c.MaxIdleConnections != 0
}

// getMaxIdleConnectionsOrDefault returns the MaxIdleConnections or the default value if not set.
func (c *ClientConfig) getMaxIdleConnectionsOrDefault() int {
	if c.MaxIdleConnections == 0 {
		// Not set, use the default.
		return defaultMaxIdleConnections
	}
	if c.MaxIdleConnections < 0 {
		// No limit.
		return 0
	}
	return c.MaxIdleConnections
}
