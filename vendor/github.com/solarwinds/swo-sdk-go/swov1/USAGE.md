<!-- Start SDK Example Usage [usage] -->
```go
package main

import (
	"context"
	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := swov1.New(
		swov1.WithSecurity(os.Getenv("SWO_API_TOKEN")),
	)

	res, err := s.ChangeEvents.CreateChangeEvent(ctx, components.ChangeEvent{
		ID:        swov1.Int64(1731676626),
		Name:      "app-deploys",
		Title:     "deployed v45",
		Timestamp: swov1.Int64(1731676626),
		Source:    swov1.String("foo3.example.com"),
		Tags: map[string]string{
			"app":         "foo",
			"environment": "production",
		},
		Links: []components.CommonLink{
			components.CommonLink{
				Rel:  "self",
				Href: "https://example.com",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Object != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->