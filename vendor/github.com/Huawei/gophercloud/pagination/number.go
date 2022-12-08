package pagination

import (
	"fmt"
	"reflect"
	"github.com/Huawei/gophercloud"
)

// NumberPage is a stricter Page interface that describes additional functionality required for use with NewNumberPage.
// For convenience, embed the NumberPageBase struct.
type NumberPage interface {
	Page

	LastStartNumber() (string, error)
}

// NumberPageBase is a page in a collection that's paginated by "limit" and "start_number" query parameters.
type NumberPageBase struct {
	PageResult

	Owner NumberPage
}

// NextPageURL generates the URL for the page of results after this one.
func (current NumberPageBase) NextPageURL() (string, error) {
	currentURL := current.URL

	startNumber, err := current.Owner.LastStartNumber()
	if err != nil {
		return "", err
	}

	if startNumber == "" {
		return "", err
	}

	q := currentURL.Query()
	q.Set("start_number", startNumber)
	currentURL.RawQuery = q.Encode()

	return currentURL.String(), nil
}

// IsEmpty satisifies the IsEmpty method of the Page interface
func (current NumberPageBase) IsEmpty() (bool, error) {
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
func (current NumberPageBase) GetBody() interface{} {
	return current.Body
}
