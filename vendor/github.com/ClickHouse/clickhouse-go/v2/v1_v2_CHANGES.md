# Breaking Changes v1 to v2

Known breaking changes for v1 to v2 are collated below. These are subject to change, if a fix is possible, and reflect the latest release only.

- v1 allowed precision loss when inserting types. For example, a sql.NullInt32 could be inserted to a UInt8 column and float64 and Decimals were interchangeable. Whilst v2 aims to be flexible, it will not transparently loose precision. Users must accept and explicitly perform this work outside the client.
- strings cannot be inserted in Date or DateTime columns in v2. [#574](https://github.com/ClickHouse/clickhouse-go/issues/574)
- Arrays must be strongly typed in v2 e.g. a `[]any` containing strings cannot be inserted into a string column. This conversion must be down outside the client, since it incurs a cost - the array must be iterated and converted. The client will not conceal this overhead in v2.
- v1 used a connection strategy of random. v2 uses in_order by default.
