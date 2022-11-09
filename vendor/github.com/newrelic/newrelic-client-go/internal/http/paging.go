package http

import (
	"net/http"

	"github.com/tomnomnom/linkheader"
)

// Pager represents a pagination implementation.
type Pager interface {
	Parse(res *http.Response) Paging
}

// Paging represents the pagination context returned from the Pager implementation.
type Paging struct {
	Next string
}

// LinkHeaderPager represents a pagination implementation that adheres to RFC 5988.
type LinkHeaderPager struct{}

// Parse is used to parse a pagination context from an HTTP response.
func (l *LinkHeaderPager) Parse(resp *http.Response) Paging {
	paging := Paging{}
	header := resp.Header.Get("Link")
	if header != "" {
		links := linkheader.Parse(header)

		for _, link := range links.FilterByRel("next") {
			paging.Next = link.URL
			break
		}
	}

	return paging
}
