package pagination

import (
	"fmt"
	"reflect"
	"github.com/Huawei/gophercloud"
	"strconv"
)

// OffsetPage is a page in a collection that's paginated by "limit" and "offset" query parameters.
type Offset struct {
	PageResult

}

// NextPageURL always returns "" to indicate that there are no more pages to return.
func (current Offset) NextPageURL() (string, error) {
	currentURL := current.URL

	q := currentURL.Query()
	currentOffset := q.Get("offset")
	currentLimit := q.Get("limit")

	/*currentCount := q.Get("total_count")
	count,err := strconv.Atoi(currentCount)
	if err != nil {
		return "", err
	}
	*/
	if currentOffset == "" {
		return "", nil
	}

	offset,err := strconv.Atoi(currentOffset)
	if err != nil {
		return "", err
	}

	if currentLimit == "" {
		return "", nil
	}

	limit,err := strconv.Atoi(currentLimit)

	if err != nil {
		return "", err
	}

	q.Set("offset", strconv.Itoa(offset + limit))
	currentURL.RawQuery = q.Encode()

	return currentURL.String(), nil
}

// IsEmpty satisfies the IsEmpty method of the Page interface
func (current Offset) IsEmpty() (bool, error) {
	if b, ok := current.Body.([]interface{}); ok {
		return len(b) == 0, nil
	}
	expected := "[]interface{}"
	actual := fmt.Sprintf("%v", reflect.TypeOf(current.Body))
	message := fmt.Sprintf(gophercloud.CE_ErrUnexpectedTypeMessage, expected, actual)
	err := gophercloud.NewSystemCommonError(gophercloud.CE_ErrUnexpectedTypeCode, message)
	return true, err
}

// GetBody returns the linked page's body. This method is needed to satisfy the
// Page interface.
func (current Offset) GetBody() interface{} {
	return current.Body
}