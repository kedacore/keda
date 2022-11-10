package metricdata

import (
	"github.com/Huawei/gophercloud"
)

type MetricData struct {
	// Specifies the namespace in service.
	Namespace string `json:"namespace"`

	// The value can be a string of 1 to 64 characters
	// and must start with a letter and contain only uppercase
	// letters, lowercase letters, digits, and underscores.
	MetricName string `json:"metric_name"`

	//Specifies the list of the metric dimensions.
	Dimensions []map[string]interface{} `json:"dimensions"`

	// Specifies the metric data list.
	Datapoints []map[string]interface{} `json:"datapoints"`

	// Specifies the metric unit.
	Unit string `json:"unit"`
}

type MetricDatasResult struct {
	gophercloud.Result
}

// ExtractMetricDatas is a function that accepts a result and extracts metric datas.
func (r MetricDatasResult) ExtractMetricDatas() ([]MetricData, error) {
	var s struct {
		// Specifies the metric data.
		MetricDatas []MetricData `json:"metrics"`
	}
	err := r.ExtractInto(&s)
	return s.MetricDatas, err
}

type Datapoint struct {
	// 指标值，该字段名称与请求参数中filter使用的查询值相同。
	Average float64 `json:"average"`
	// 指标采集时间。
	Timestamp int `json:"timestamp"`
	// 指标单位
	Unit string `json:"unit,omitempty"`
}

type EventDataInfo struct {
	// 事件类型，例如instance_host_info。
	Type string `json:"type"`
	// 事件上报时间。
	Timestamp int `json:"timestamp"`
	// 主机配置信息。
	Value string `json:"value"`
}

// This is a auto create Response Object
type EventData struct {
	Datapoints []EventDataInfo `json:"datapoints"`
}

type Metricdata struct {
	//  指标数据列表。由于查询数据时，云监控会根据所选择的聚合粒度向前取整from参数，所以datapoints中包含的数据点有可能会多于预期。
	Datapoints []Datapoint `json:"datapoints"`
	// 指标名称，例如弹性云服务器监控指标中的cpu_util。
	MetricName string `json:"metric_name"`
}

type AddMetricDataResult struct {
	gophercloud.ErrResult
}

type GetEventDataResult struct {
	gophercloud.Result
}

type GetResult struct {
	gophercloud.Result
}

func (r GetEventDataResult) Extract() (*EventData, error) {
	var s *EventData
	err := r.ExtractInto(&s)
	return s, err
}

func (r GetResult) Extract() (*Metricdata, error) {
	var s *Metricdata
	err := r.ExtractInto(&s)
	return s, err
}
