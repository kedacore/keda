package v1

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-kusto-go/kusto/internal/frames/unmarshal"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames/unmarshal/json"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
)

// Reference: This is what the top level data structure looks like for a V1 query. However, we are
// not using it because we want to stream the DataTable(s) back instead of reading all into memory.
/*
type DataSet struct {
	Tables []DataTable
}
*/

// Decoder implements frames.Decoder on the REST v1 frames.
type Decoder struct {
	dec *json.Decoder
	op  errors.Op
}

// Decode implements frames.Decoder.Decode(). This is not thread safe.
func (d *Decoder) Decode(ctx context.Context, r io.ReadCloser, op errors.Op) chan frames.Frame {
	ch := make(chan frames.Frame, 1) // Channel is sized to 1. We read from the channel faster than we put on the channel.
	d.dec = json.NewDecoder(r)
	d.op = op

	go func() {
		defer r.Close()
		defer close(ch)

		if err := d.nextDelimEquals('{'); err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}

		if err := d.findStringToken("Tables"); err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}

		if err := d.nextDelimEquals('['); err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}

		if err := d.processTables(ctx, ch); err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}

		if err := d.nextDelimEquals(']'); err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}

		if err := d.nextDelimEquals('}'); err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}
	}()
	return ch
}

func (d *Decoder) nextDelimEquals(r rune) error {
	t, err := d.dec.Token()
	if err != nil {
		return fmt.Errorf("(v1)could not get the %s token: %s", string(r), err)
	}
	if t != json.Delim(r) {
		return fmt.Errorf("(v1)did not get the expected token, got %q, want %q", t, string(r))
	}
	return nil
}

// findStringToken looks within the current delimiter for a string token.
func (d *Decoder) findStringToken(s string) error {
	for {
		if !d.dec.More() {
			return fmt.Errorf("(v1)could not find the %q token within a message", s)
		}

		t, err := d.dec.Token()
		if err != nil {
			return fmt.Errorf("(v1)could not get the Tables entry token: %s", err)
		}
		if v, ok := t.(string); ok {
			if v == s {
				return nil
			}
		}
	}
	panic("unreachable")
}

func (d *Decoder) processTables(ctx context.Context, ch chan frames.Frame) error {
	rows := unmarshal.GetRows()
	defer unmarshal.PutRows(rows)

	for d.dec.More() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		dt := DataTable{Rows: rows, Op: d.op}

		err := d.dec.Decode(&dt)
		if err != nil {
			return err
		}

		columns, err := dt.DataTypes.ToColumns()
		if err != nil {
			return err
		}

		dt.KustoRows, err = unmarshal.Rows(columns, dt.Rows)
		if err != nil {
			return err
		}
		dt.Rows = nil

		ch <- dt
	}
	return nil
}
