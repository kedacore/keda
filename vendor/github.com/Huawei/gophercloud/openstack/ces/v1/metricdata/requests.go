package metricdata

import (
	"github.com/Huawei/gophercloud"
)

// BatchQueryOptsBuilder allows extensions to add additional parameters to the
// BatchQuery request.
type BatchQueryOptsBuilder interface {
	ToBatchQueryOptsMap() (map[string]interface{}, error)
}

type Metric struct {
	// Specifies the namespace in service.
	Namespace string `json:"namespace" required:"true"`

	// The value can be a string of 1 to 64 characters
	// and must start with a letter and contain only uppercase
	// letters, lowercase letters, digits, and underscores.
	MetricName string `json:"metric_name" required:"true"`

	// Specifies the list of the metric dimensions.
	Dimensions []map[string]string `json:"dimensions" required:"true"`
}

// BatchQueryOpts represents options for batch query metric data.
type BatchQueryOpts struct {
	// Specifies the metric data.
	Metrics []Metric `json:"metrics" required:"true"`

	// Specifies the start time of the query.
	From int64 `json:"from" required:"true"`

	// Specifies the end time of the query.
	To int64 `json:"to" required:"true"`

	// Specifies the data monitoring granularity.
	Period string `json:"period" required:"true"`

	// Specifies the data rollup method.
	Filter string `json:"filter" required:"true"`
}

// ToBatchQueryOptsMap builds a request body from BatchQueryOpts.
func (opts BatchQueryOpts) ToBatchQueryOptsMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "")
}

// Querying Monitoring Data in Batches.
func BatchQuery(client *gophercloud.ServiceClient, opts BatchQueryOptsBuilder) (r MetricDatasResult) {
	b, err := opts.ToBatchQueryOptsMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = client.Post(batchQueryMetricDataURL(client), b, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200}})
	return
}

type AddMetricDataOpts []AddMetricDataItem

type GetEventDataOpts struct {
	// 指标的维度，目前最大支持3个维度，维度编号从0开始；维度格式为dim.{i}=key,value参考弹性云服务器维度。例如dim.0=instance_id,i-12345
	Dim0 string `q:"dim.0,required"`
	Dim1 string `q:"dim.1"`
	Dim2 string `q:"dim.2"`
	// 查询数据起始时间，UNIX时间戳，单位毫秒。
	From string `q:"from,required"`
	// 指标命名空间，例如弹性云服务器命名空间。
	Namespace string `q:"namespace,required"`
	// 查询数据截止时间UNIX时间戳，单位毫秒。from必须小于to。
	To string `q:"to,required"`
	// 事件类型，只允许字母、下划线、中划线，字母开头，长度不超过64，如instance_host_info。
	Type string `q:"type,required"`
}

type GetOpts struct {
	// 指标的维度，目前最大支持3个维度，维度编号从0开始；维度格式为dim.{i}=key,value，最大值为256。  例如dim.0=instance_id,i-12345
	Dim0 string `q:"dim.0,required"`
	Dim1 string `q:"dim.1"`
	Dim2 string `q:"dim.2"`
	// 数据聚合方式。  支持的值为max, min, average, sum, variance。
	Filter string `q:"filter,required"`
	// 查询数据起始时间，UNIX时间戳，单位毫秒。建议from的值相对于当前时间向前偏移至少1个周期。由于聚合运算的过程是将一个聚合周期范围内的数据点聚合到周期起始边界上，如果将from和to的范围设置在聚合周期内，会因为聚合未完成而造成查询数据为空，所以建议from参数相对于当前时间向前偏移至少1个周期。以5分钟聚合周期为例：假设当前时间点为10:35，10:30~10:35之间的原始数据会被聚合到10:30这个点上，所以查询5分钟数据点时from参数应为10:30或之前。云监控会根据所选择的聚合粒度向前取整from参数。
	From string `q:"from,required"`
	// 指标名称，例如弹性云服务器监控指标中的cpu_util。
	MetricName string `q:"metric_name,required"`
	// 指标命名空间。
	Namespace string `q:"namespace,required"`
	// 监控数据粒度。  取值范围：  1，实时数据 300，5分钟粒度 1200，20分钟粒度 3600，1小时粒度 14400，4小时粒度 86400，1天粒度
	Period string `q:"period,required"`
	// 查询数据截止时间UNIX时间戳，单位毫秒。from必须小于to。
	To string `q:"to,required"`
}

type AddMetricDataItem struct {
	// 指标数据。
	Metric MetricInfo `json:"metric" required:"true"`
	// 数据的有效期，超出该有效期则自动删除该数据，单位秒，最大值604800。
	Ttl int `json:"ttl" required:"true"`
	// 数据收集时间  UNIX时间戳，单位毫秒。  说明： 因为客户端到服务器端有延时，因此插入数据的时间戳应该在[当前时间-3天+20秒，当前时间+10分钟-20秒]区间内，保证到达服务器时不会因为传输时延造成数据不能插入数据库。
	CollectTime int `json:"collect_time" required:"true"`
	// 指标数据的值。
	Value float64 `json:"value" required:"true"`
	// 数据的单位。
	Unit string `json:"unit,omitempty"`
	// 数据的类型，只能是\"int\"或\"float\"
	Type string `json:"type,omitempty"`
}

// 指标信息
type MetricInfo struct {
	// 指标维度
	Dimensions []MetricsDimension `json:"dimensions" required:"true"`
	// 指标名称，必须以字母开头，只能包含0-9/a-z/A-Z/_，长度最短为1，最大为64。  具体指标名请参见查询指标列表中查询出的指标名。
	MetricName string `json:"metric_name" required:"true"`
	// 指标命名空间，，例如弹性云服务器命名空间。格式为service.item；service和item必须是字符串，必须以字母开头，只能包含0-9/a-z/A-Z/_，总长度最短为3，最大为32。说明： 当alarm_type为（EVENT.SYS| EVENT.CUSTOM）时允许为空。
	Namespace string `json:"namespace" required:"true"`
}

// 指标维度
type MetricsDimension struct {
	// 维度名
	Name string `json:"name,omitempty"`
	// 维度值
	Value string `json:"value,omitempty"`
}

func (opts AddMetricDataItem) ToMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "")
}

type AddMetricDataOptsBuilder interface {
	ToAddMetricDataMap() ([]map[string]interface{}, error)
}

func (opts AddMetricDataOpts) ToAddMetricDataMap() ([]map[string]interface{}, error) {
	newOpts := make([]map[string]interface{}, len(opts))
	for i, opt := range opts {
		opt, err := opt.ToMap()
		if err != nil {
			return nil, err
		}
		newOpts[i] = opt
	}
	return newOpts, nil
}

func AddMetricData(client *gophercloud.ServiceClient, opts AddMetricDataOptsBuilder) (r AddMetricDataResult) {
	b, err := opts.ToAddMetricDataMap()
	if err != nil {
		r.Err = err
		return
	}

	_, r.Err = client.Post(addMetricDataURL(client), b, nil, &gophercloud.RequestOpts{
		OkCodes: []int{201},
	})
	return
}

func GetEventData(client *gophercloud.ServiceClient, opts GetEventDataOpts) (r GetEventDataResult) {
	q, err := gophercloud.BuildQueryString(&opts)
	if err != nil {
		r.Err = err
		return
	}
	url := getEventDataURL(client) + q.String()
	_, r.Err = client.Get(url, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200},
	})

	return
}

func Get(client *gophercloud.ServiceClient, opts GetOpts) (r GetResult) {
	q, err := gophercloud.BuildQueryString(&opts)
	if err != nil {
		r.Err = err
		return
	}
	url := getURL(client) + q.String()
	_, r.Err = client.Get(url, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200},
	})

	return
}
