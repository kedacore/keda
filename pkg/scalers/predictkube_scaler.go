package scalers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/valyala/fasthttp"
	"github.com/xhit/go-str2duration/v2"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	predictKubeAddress    = "https://forecaster.ais.dysnix.org/predict"
	predictKubeMetricType = "External"
)

var (
	defaultStep = time.Minute * 5
)

type PredictRequest struct {
	ForecastHorizon uint           `json:"forecastHorizon"`
	Observations    []*MetricValue `json:"observations"`
}

type PredictResponse struct {
	Predictions []*MetricValue `json:"predictions"`
}

func (p PredictResponse) Len() int {
	return len(p.Predictions)
}

func (p PredictResponse) Less(i, j int) bool {
	return p.Predictions[i].Date.Before(p.Predictions[j].Date)
}

func (p PredictResponse) Swap(i, j int) {
	p.Predictions[i], p.Predictions[j] = p.Predictions[j], p.Predictions[i]
}

type MetricValue struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"rps"`
}

func (mv *MetricValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Date  string  `json:"date"`
		Value float64 `json:"rps"`
	}{
		Date: fmt.Sprintf("%02d-%02d-%d %02d:%02d:%02d",
			mv.Date.Month(), mv.Date.Day(), mv.Date.Year(),
			mv.Date.Hour(), mv.Date.Minute(), mv.Date.Second()),
		Value: mv.Value,
	})
}

func (mv *MetricValue) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		Date  string  `json:"date"`
		Value float64 `json:"rps"`
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if mv == nil {
		*mv = MetricValue{}
	}

	if len(tmp.Date) == 19 {
		month, err := strconv.Atoi(tmp.Date[0:2])
		if err != nil {
			return err
		}

		day, err := strconv.Atoi(tmp.Date[3:5])
		if err != nil {
			return err
		}

		year, err := strconv.Atoi(tmp.Date[6:10])
		if err != nil {
			return err
		}

		hour, err := strconv.Atoi(tmp.Date[11:13])
		if err != nil {
			return err
		}

		minutes, err := strconv.Atoi(tmp.Date[14:16])
		if err != nil {
			return err
		}

		seconds, err := strconv.Atoi(tmp.Date[17:19])
		if err != nil {
			return err
		}

		mv.Date = time.Date(year, time.Month(month), day, hour, minutes, seconds, 0, time.UTC)
	}

	mv.Value = tmp.Value

	return nil
}

type predictKubeScaler struct {
	metadata         *predictKubeMetadata
	prometheusClient api.Client
	httpClient       *http.Client
	api              v1.API
	transport        *transport
}

type predictKubeMetadata struct {
	predictHorizon    time.Duration
	historyTimeWindow time.Duration
	stepDuration      time.Duration
	apiKey            string
	predictKubeSite   string
	prometheusAddress string
	metricName        string
	query             string
	threshold         int64
	scalerIndex       int
}

var predictKubeLog = logf.Log.WithName("predictkube_scaler")

// NewPredictKubeScaler creates a new PredictKube scaler
func NewPredictKubeScaler(ctx context.Context, config *ScalerConfig) ( /*Scaler*/ *predictKubeScaler, error) {
	s := &predictKubeScaler{
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				MaxIdleConns:    10,
				WriteBufferSize: 128 << 10,
				ReadBufferSize:  128 << 10,
			},
		},
	}

	meta, err := parsePredictKubeMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing PredictKube metadata: %s", err)
	}

	s.metadata = meta

	err = s.initPredictKubePrometheusConn(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error create Prometheus client and API objects: %s", err)
	}
	return s, nil
}

// IsActive returns true if we are able to get metrics from PredictKube
func (pks *predictKubeScaler) IsActive(ctx context.Context) (bool, error) {
	//return true, pks.ping(ctx) // TODO: ???
	return true, nil
}

func (pks *predictKubeScaler) Close(_ context.Context) error {
	pks.transport.close()
	pks.httpClient.CloseIdleConnections()
	return nil
}

func (pks *predictKubeScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(pks.metadata.threshold, resource.DecimalSI)
	metricName := kedautil.NormalizeString(fmt.Sprintf("predictkube-%s", pks.metadata.metricName))
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(pks.metadata.scalerIndex, metricName),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: predictKubeMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

func (pks *predictKubeScaler) GetMetrics(ctx context.Context, metricName string, _ labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := pks.doPredictRequest(ctx)
	if err != nil {
		predictKubeLog.Error(err, "error executing query to predict controller service")
		return []external_metrics.ExternalMetricValue{}, err
	}

	var value int64
	if val != nil && len(val.Predictions) > 0 {
		lastElement := val.Predictions[len(val.Predictions)-1]
		if lastElement != nil {
			value = int64(lastElement.Value)
		}
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(value, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (pks *predictKubeScaler) doPredictRequest(ctx context.Context) (*PredictResponse, error) {
	results, err := pks.doQuery(ctx)
	if err != nil {
		return nil, err
	}

	return pks.doRequest(ctx, &PredictRequest{
		ForecastHorizon: uint(math.Round(float64(pks.metadata.historyTimeWindow / pks.metadata.stepDuration))),
		Observations:    results,
	})
}

func (pks *predictKubeScaler) doRequest(ctx context.Context, data *PredictRequest) (*PredictResponse, error) {
	bts, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", pks.metadata.predictKubeSite, bytes.NewReader(bts))
	if err != nil {
		return nil, err
	}

	resp, err := pks.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	result := &PredictResponse{}
	err = json.NewDecoder(resp.Body).Decode(result)

	if err != nil {
		return nil, err
	}

	sort.Sort(result)

	return result, nil
}

func (pks *predictKubeScaler) doQuery(ctx context.Context) ([]*MetricValue, error) {
	currentTime := time.Now()

	// TODO: parse from query...
	if pks.metadata.stepDuration == 0 {
		pks.metadata.stepDuration = defaultStep
	}

	r := v1.Range{
		Start: currentTime.Add(-pks.metadata.historyTimeWindow),
		End:   currentTime,
		Step:  pks.metadata.stepDuration,
	}

	val, warns, err := pks.api.QueryRange(ctx, pks.metadata.query, r)

	if len(warns) > 0 {
		predictKubeLog.V(1).Info("warnings", warns)
	}

	if err != nil {
		return nil, err
	}

	return pks.parsePrometheusResult(val)
}

func (pks *predictKubeScaler) parsePrometheusResult(result model.Value) (out []*MetricValue, err error) {
	switch result.Type() {
	case model.ValVector:
		if res, ok := result.(model.Vector); ok {
			for _, val := range res {
				out = append(out, &MetricValue{
					Date:  val.Timestamp.Time(),
					Value: float64(val.Value),
				})
			}
		}
	case model.ValMatrix:
		if res, ok := result.(model.Matrix); ok {
			for _, val := range res {
				for _, v := range val.Values {
					out = append(out, &MetricValue{
						Date:  v.Timestamp.Time(),
						Value: float64(v.Value),
					})
				}
			}
		}
	case model.ValScalar:
		if res, ok := result.(*model.Scalar); ok {
			out = append(out, &MetricValue{
				Date:  res.Timestamp.Time(),
				Value: float64(res.Value),
			})
		}
	case model.ValString:
		if res, ok := result.(*model.String); ok {
			s, err := strconv.ParseFloat(res.Value, 64)
			if err != nil {
				return nil, err
			}

			out = append(out, &MetricValue{
				Date:  res.Timestamp.Time(),
				Value: s,
			})
		}
	}

	return out, nil
}

func parsePredictKubeMetadata(config *ScalerConfig) (result *predictKubeMetadata, err error) {
	meta := predictKubeMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["prometheusAddress"]; ok {
		meta.prometheusAddress = val
	} else {
		return nil, fmt.Errorf("no prometheusAddress given")
	}

	if val, ok := config.TriggerMetadata["predictHorizon"]; ok {
		meta.predictHorizon, err = str2duration.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("predictHorizon parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no predictHorizon given")
	}

	if val, ok := config.TriggerMetadata["queryStep"]; ok {
		meta.stepDuration, err = str2duration.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("queryStep parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no queryStep given")
	}

	if val, ok := config.TriggerMetadata["historyTimeWindow"]; ok {
		meta.historyTimeWindow, err = str2duration.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("historyTimeWindow parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no historyTimeWindow given")
	}

	if val, ok := config.TriggerMetadata["threshold"]; ok {
		meta.threshold, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("threshold parsing error %s", err.Error())
		}
	} else {
		return nil, fmt.Errorf("no threshold given")
	}

	if val, ok := config.TriggerMetadata["metricName"]; ok {
		meta.metricName = val //GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("predictkube-%s", val)))
	} else {
		return nil, fmt.Errorf("no metric name given")
	}

	meta.scalerIndex = config.ScalerIndex

	// TODO: check AuthParams...
	if val, ok := config.AuthParams["apiKey"]; ok {
		meta.apiKey = val
	} else {
		return nil, fmt.Errorf("no api key given")
	}

	if val, ok := config.AuthParams["predictKubeSite"]; ok {
		if val != "" {
			meta.predictKubeSite = val
		} else {
			meta.predictKubeSite = predictKubeAddress
		}
	} else {
		meta.predictKubeSite = predictKubeAddress
	}

	return &meta, nil
}

func (pks *predictKubeScaler) ping(ctx context.Context) (err error) {
	_, err = pks.api.Runtimeinfo(ctx)
	return err
}

// initPredictKubePrometheusConn init prometheus client and setup connection to API
func (pks *predictKubeScaler) initPredictKubePrometheusConn(ctx context.Context, meta *predictKubeMetadata) (err error) {
	pks.transport = newTransport(&httpTransport{
		maxIdleConnDuration: 10,
		readTimeout:         time.Second * 15,
		writeTimeout:        time.Second * 15,
	})

	pks.prometheusClient, err = api.NewClient(api.Config{
		Address:      pks.metadata.prometheusAddress,
		RoundTripper: pks.transport,
	})

	if err != nil {
		predictKubeLog.V(1).Error(err, "init Prometheus client")
		return err
	}

	pks.api = v1.NewAPI(pks.prometheusClient)

	if isNotTest := ctx.Value("is_not_test"); isNotTest != nil {
		if t, ok := isNotTest.(bool); ok && t {
			err = pks.ping(context.Background())
		}
	}

	return err
}

type httpTransport struct {
	maxIdleConnDuration time.Duration
	readTimeout         time.Duration
	writeTimeout        time.Duration
}

// transport implements the estransport interface with
// the github.com/valyala/fasthttp HTTP client.
type transport struct {
	client *fasthttp.Client
}

func newTransport(opts *httpTransport) *transport {
	return &transport{
		client: &fasthttp.Client{
			MaxIdleConnDuration: opts.maxIdleConnDuration,
			ReadTimeout:         opts.readTimeout,
			WriteTimeout:        opts.writeTimeout,
			// nolint:gosec
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// RoundTrip performs the request and returns a response or error
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	freq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(freq)

	fres := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fres)

	copyRequest(freq, req)

	err := t.client.Do(freq, fres)
	if err != nil {
		return nil, err
	}

	res := &http.Response{Header: make(http.Header)}
	copyResponse(res, fres)

	return res, nil
}

func (t *transport) close() {
	t.client.CloseIdleConnections()
}

// copyRequest converts a http.Request to fasthttp.Request
func copyRequest(dst *fasthttp.Request, src *http.Request) {
	if src.Method == fasthttp.MethodGet && src.Body != nil {
		src.Method = fasthttp.MethodPost
	}

	dst.SetHost(src.Host)
	dst.SetRequestURI(src.URL.String())

	dst.Header.SetRequestURI(src.URL.String())
	dst.Header.SetMethod(src.Method)

	for k, vv := range src.Header {
		for _, v := range vv {
			dst.Header.Set(k, v)
		}
	}

	if src.Body != nil {
		dst.SetBodyStream(bodyCloserReader{
			body: src.Body,
		}, -1)
	}
}

// copyResponse converts a http.Response to fasthttp.Response
func copyResponse(dst *http.Response, src *fasthttp.Response) {
	dst.StatusCode = src.StatusCode()

	src.Header.VisitAll(func(k, v []byte) {
		dst.Header.Set(string(k), string(v))
	})

	// Cast to a string to make a copy seeing as src.Body() won't
	// be valid after the response is released back to the pool (fasthttp.ReleaseResponse).
	dst.Body = ioutil.NopCloser(strings.NewReader(string(src.Body())))
}

type bodyCloserReader struct {
	body io.ReadCloser
}

func (bcr bodyCloserReader) Read(p []byte) (n int, err error) {
	n, err = bcr.body.Read(p)

	if err != nil {
		_ = bcr.body.Close()
	}

	return n, err
}
