package pagination

import (
	"fmt"
	"reflect"
	"github.com/Huawei/gophercloud"
	"strconv"
)

// OffsetPage is a page in a collection that's paginated by "limit" and "offset" query parameters.
type OffsetPage struct {
	PageResult

}

// NextPageURL always returns "" to indicate that there are no more pages to return.
func (current OffsetPage) NextPageURL() (string, error) {
	currentURL := current.URL

	q := currentURL.Query()
	currentOffset := q.Get("offset")

	if currentOffset == "" {
		return "", nil
	}

	offset,err := strconv.Atoi(currentOffset)
	if err != nil {
		return "", err
	}

	q.Set("offset", strconv.Itoa(offset + 1))
	currentURL.RawQuery = q.Encode()

	return currentURL.String(), nil
}

// IsEmpty satisfies the IsEmpty method of the Page interface
func (current OffsetPage) IsEmpty() (bool, error) {
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
func (current OffsetPage) GetBody() interface{} {
	return current.Body
}