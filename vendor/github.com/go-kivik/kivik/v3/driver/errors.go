package driver

type err string

func (e err) Error() string {
	return string(e)
}

// EOQ should be returned by a view iterator at the end of each query result
// set.
const EOQ = err("EOQ")
