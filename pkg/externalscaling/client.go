package externalscaling

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	cl "github.com/kedacore/keda/v2/pkg/externalscaling/api"
)

type GrpcClient struct {
	Client     cl.ExternalCalculationClient
	Connection *grpc.ClientConn
}

func NewGrpcClient(url string, certDir string) (*GrpcClient, error) {
	retryPolicy := `{
		"methodConfig": [{
		  "timeout": "3s",
		  "waitForReady": true,
		  "retryPolicy": {
			  "InitialBackoff": ".25s",
			  "MaxBackoff": "2.0s",
			  "BackoffMultiplier": 2,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }
		}]}`

	opts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(retryPolicy),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// if certDir is not empty, load certificates
	if certDir != "" {
		creds, err := loadCertificates(certDir)
		if err != nil {
			return nil, fmt.Errorf("externalCalculator error while creating new client: %w", err)
		}
		opts = []grpc.DialOption{
			grpc.WithDefaultServiceConfig(retryPolicy),
			grpc.WithTransportCredentials(creds),
		}
	}

	conn, err := grpc.Dial(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("externalCalculator error while creating new client: %w", err)
	}

	return &GrpcClient{Client: cl.NewExternalCalculationClient(conn), Connection: conn}, nil
}

func (c *GrpcClient) Calculate(ctx context.Context, list *cl.MetricsList) (*cl.MetricsList, error) {
	response, err := c.Client.Calculate(ctx, list)
	if err != nil {
		return nil, fmt.Errorf("error in externalscaling.Calculate %w", err)
	}
	return response.List, nil
}

// WaitForConnectionReady waits for gRPC connection to be ready
// returns true if the connection was successful, false if we hit a timeout or context canceled
func (c *GrpcClient) WaitForConnectionReady(ctx context.Context, url string, timeout time.Duration, logger logr.Logger) bool {
	currentState := c.Connection.GetState()
	if currentState != connectivity.Ready {
		logger.Info(fmt.Sprintf("Waiting for %v to establish a gRPC connection to server for external calculator at %s", timeout, url))
		timeoutTimer := time.After(timeout)
		for {
			select {
			case <-ctx.Done():
				return false
			case <-timeoutTimer:
				err := fmt.Errorf("hit '%v' timeout trying to connect externalCalculator at '%s'", timeout, url)
				logger.Error(err, "error while waiting for connection for externalCalculator")
				return false
			default:
				c.Connection.Connect()
				time.Sleep(500 * time.Millisecond)
				currentState := c.Connection.GetState()
				if currentState == connectivity.Ready {
					return true
				}
			}
		}
	}
	return true
}

// ConvertToGeneratedStruct converts K8s external metrics list to gRPC generated
// external metrics list
func ConvertToGeneratedStruct(inK8sList []external_metrics.ExternalMetricValue) *cl.MetricsList {
	outExternal := cl.MetricsList{}
	for _, val := range inK8sList {
		metric := cl.Metric{Name: val.MetricName, Value: float32(val.Value.Value())}
		outExternal.MetricValues = append(outExternal.MetricValues, &metric)
	}
	return &outExternal
}

// ConvertFromGeneratedStruct converts gRPC generated external metrics list to
// K8s external_metrics list
func ConvertFromGeneratedStruct(inExternal *cl.MetricsList) []external_metrics.ExternalMetricValue {
	outK8sList := []external_metrics.ExternalMetricValue{}
	for _, inValue := range inExternal.MetricValues {
		outValue := external_metrics.ExternalMetricValue{}
		outValue.MetricName = inValue.Name
		outValue.Timestamp = v1.Now()
		outValue.Value.SetMilli(int64(inValue.Value * 1000))
		outK8sList = append(outK8sList, outValue)
	}
	return outK8sList
}

// close connection
func (c *GrpcClient) CloseConnection() error {
	if c.Connection != nil {
		err := c.Connection.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// load certificates taken from a directory given as an argument
// expects ca.crt, tls.crt and tls.key to be present in the directory
func loadCertificates(certDir string) (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed client's certificate
	pemClientCA, err := os.ReadFile(path.Join(certDir, "ca.crt"))
	if err != nil {
		return nil, err
	}

	// Get the SystemCertPool, continue with an empty pool on error
	certPool, _ := x509.SystemCertPool()
	if certPool == nil {
		certPool = x509.NewCertPool()
	}
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	// Load certificate and private key
	cert, err := tls.LoadX509KeyPair(path.Join(certDir, "tls.crt"), path.Join(certDir, "tls.key"))
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
	}
	config.RootCAs = certPool

	return credentials.NewTLS(config), nil
}
