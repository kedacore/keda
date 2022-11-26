package metricdata

import "github.com/Huawei/gophercloud"

// batch query metric data url
func batchQueryMetricDataURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("batch-query-metric-data")
}

func addMetricDataURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("metric-data")
}

func getEventDataURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("event-data")
}

func getURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("metric-data")
}