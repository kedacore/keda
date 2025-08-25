package sumologic

import "time"

type Query struct {
	Type             string
	Query            string
	Queries          map[string]string
	ResultQueryRowID string
	Quantization     time.Duration
	Rollup           string
	ResultField      string
	TimeRange        time.Duration
	Timezone         string
	Aggregator       string
}

type QueryBuilder struct {
	query Query
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: Query{},
	}
}

func (qb *QueryBuilder) Type(qtype string) *QueryBuilder {
	qb.query.Type = qtype
	return qb
}

func (qb *QueryBuilder) Query(query string) *QueryBuilder {
	qb.query.Query = query
	return qb
}

func (qb *QueryBuilder) Queries(queries map[string]string) *QueryBuilder {
	qb.query.Queries = queries
	return qb
}

func (qb *QueryBuilder) ResultQueryRowID(resultQueryRowID string) *QueryBuilder {
	qb.query.ResultQueryRowID = resultQueryRowID
	return qb
}

func (qb *QueryBuilder) Quantization(qtz time.Duration) *QueryBuilder {
	qb.query.Quantization = qtz
	return qb
}

func (qb *QueryBuilder) Rollup(rollup string) *QueryBuilder {
	qb.query.Rollup = rollup
	return qb
}

func (qb *QueryBuilder) ResultField(resultField string) *QueryBuilder {
	qb.query.ResultField = resultField
	return qb
}

func (qb *QueryBuilder) TimeRange(timerange time.Duration) *QueryBuilder {
	qb.query.TimeRange = timerange
	return qb
}

func (qb *QueryBuilder) Timezone(timezone string) *QueryBuilder {
	qb.query.Timezone = timezone
	return qb
}

func (qb *QueryBuilder) Aggregator(aggregator string) *QueryBuilder {
	qb.query.Aggregator = aggregator
	return qb
}

func (qb *QueryBuilder) Build() Query {
	return qb.query
}
