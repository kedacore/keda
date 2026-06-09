package proto

import (
	"fmt"

	chproto "github.com/ClickHouse/ch-go/proto"
)

type TableColumns struct {
	First  string
	Second string
}

func (t *TableColumns) Decode(reader *chproto.Reader, revision uint64) (err error) {
	if t.First, err = reader.Str(); err != nil {
		return err
	}
	if t.Second, err = reader.Str(); err != nil {
		return err
	}
	return nil
}

func (t *TableColumns) String() string {
	return fmt.Sprintf("first=%s, second=%s", t.First, t.Second)
}
