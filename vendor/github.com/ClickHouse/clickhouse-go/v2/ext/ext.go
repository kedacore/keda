package ext

import (
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func NewTable(name string, columns ...func(t *Table) error) (*Table, error) {
	table := &Table{
		name:  name,
		block: proto.NewBlock(),
	}
	for _, column := range columns {
		if err := column(table); err != nil {
			return nil, err
		}
	}
	return table, nil
}

type Table struct {
	name  string
	block *proto.Block
}

func (tbl *Table) Name() string {
	return tbl.name
}

func (tbl *Table) Structure() string {
	columnStructure := make([]string, 0, len(tbl.block.Columns))
	for _, c := range tbl.block.Columns {
		columnStructure = append(columnStructure, fmt.Sprintf("%v %v", c.Name(), c.Type()))
	}
	return strings.Join(columnStructure, ", ")
}

func (tbl *Table) Block() *proto.Block {
	return tbl.block
}

func (tbl *Table) Append(v ...any) error {
	return tbl.block.Append(v...)
}

func Column(name string, ct column.Type) func(t *Table) error {
	return func(tbl *Table) error {
		return tbl.block.AddColumn(name, ct)
	}
}
