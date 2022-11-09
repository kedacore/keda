package v2

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames/unmarshal/json"
)

// Decoder implements frames.Decoder on the REST v2 frames.
type Decoder struct {
	columns table.Columns
	dec     *json.Decoder
	op      errors.Op

	frameRaw json.RawMessage
}

// Decode implements frames.Decoder.Decode(). This is not thread safe.
func (d *Decoder) Decode(ctx context.Context, r io.ReadCloser, op errors.Op) chan frames.Frame {
	d.columns = nil
	d.dec = json.NewDecoder(r)
	d.dec.UseNumber()
	d.op = op

	ch := make(chan frames.Frame, 1) // Channel is sized to 1. We read from the channel faster than we put on the channel.

	go func() {
		defer r.Close()
		defer close(ch)

		// We should receive a '[' indicating the start of the JSON list of Frames.
		t, err := d.dec.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}
		if t != json.Delim('[') {
			frames.Errorf(ctx, ch, "Expected '[' delimiter")
			return
		}

		// Extract the initial Frame, a dataSetHeader.
		dsh, err := d.dataSetHeader()
		if err != nil {
			frames.Errorf(ctx, ch, "first frame had error: %s", err)
			return
		}
		ch <- dsh

		// Start decoding the rest of the frames.
		d.decodeFrames(ctx, ch)

		// Expect to recieve the end of our JSON list of frames, marked by the ']' delimiter.
		t, err = d.dec.Token()
		if err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}

		if t != json.Delim(']') {
			frames.Errorf(ctx, ch, "Expected ']' delimiter")
			return
		}
	}()

	return ch
}

// dataSetHeader decodes the byte stream into a DataSetHeader.
func (d *Decoder) dataSetHeader() (DataSetHeader, error) {
	dsh := DataSetHeader{Op: d.op}
	err := d.dec.Decode(&dsh)
	return dsh, err
}

// decodeFrames is used to decode incoming frames after the DataSetHeader has been received.
func (d *Decoder) decodeFrames(ctx context.Context, ch chan frames.Frame) {
	for d.dec.More() {
		if err := d.decode(ctx, ch); err != nil {
			frames.Errorf(ctx, ch, err.Error())
			return
		}
	}
}

var (
	ftDataTable         = []byte(frames.TypeDataTable)
	ftDataSetCompletion = []byte(frames.TypeDataSetCompletion)
	ftTableHeader       = []byte(frames.TypeTableHeader)
	ftTableFragment     = []byte(frames.TypeTableFragment)
	ftTableProgress     = []byte(frames.TypeTableProgress)
	ftTableCompletion   = []byte(frames.TypeTableCompletion)
)

func (d *Decoder) decode(ctx context.Context, ch chan frames.Frame) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err := d.dec.Decode(&d.frameRaw)
	if err != nil {
		return err
	}

	ft, err := getFrameType(d.frameRaw)
	if err != nil {
		return err
	}

	switch {
	case bytes.Equal(ft, ftDataTable):
		dt := DataTable{}
		if err := dt.UnmarshalRaw(d.frameRaw); err != nil {
			return err
		}
		dt.Op = d.op
		ch <- dt
	case bytes.Equal(ft, ftDataSetCompletion):
		dc := DataSetCompletion{}
		if err := dc.UnmarshalRaw(d.frameRaw); err != nil {
			return err
		}
		dc.Op = d.op
		ch <- dc
	case bytes.Equal(ft, ftTableHeader):
		th := TableHeader{}
		if err := th.UnmarshalRaw(d.frameRaw); err != nil {
			return err
		}
		th.Op = d.op
		d.columns = th.Columns
		ch <- th
	case bytes.Equal(ft, ftTableFragment):
		tf := TableFragment{Columns: d.columns}
		if err := tf.UnmarshalRaw(d.frameRaw); err != nil {
			return err
		}
		tf.Op = d.op
		ch <- tf
	case bytes.Equal(ft, ftTableProgress):
		tp := TableProgress{}
		if err := tp.UnmarshalRaw(d.frameRaw); err != nil {
			return err
		}
		tp.Op = d.op
		ch <- tp
	case bytes.Equal(ft, ftTableCompletion):
		tc := TableCompletion{}
		if err := tc.UnmarshalRaw(d.frameRaw); err != nil {
			return err
		}
		tc.Op = d.op
		d.columns = nil
		ch <- tc
	default:
		return fmt.Errorf("received FrameType %s, which we did not expect", ft)
	}
	return nil
}

var (
	frameType = []byte(fmt.Sprintf("%q:", frames.FieldFrameType))
	comma     = []byte(`,`)
	semicolon = []byte(`:`)
)

// var frameTypeRE = regexp.MustCompile(`"FrameType"\s*:\s*"([a-zA-Z]+)"`)

// getFrameType looks through a raw frame to extract the type of frame. This allows us to decode the frame
// without decoding to a map first.
// Note: This is a fast implementation that is benchmarked, as it is on the hot path. But it is not the
// most robust. If we get problems, we can uncomment var frameTypeRE and the code below to do this. It takes
// 5x as long, but in the scheme it won't matter.
func getFrameType(message json.RawMessage) ([]byte, error) {
	/*
		matches := frameTypeRE.FindSubmatch(message)
		if len(matches) < 2 {
			return nil, fmt.Errorf("FrameType was missing in a frame")
		}
		return matches[1], nil
	*/

	message = bytes.TrimSpace(message)
	message = bytes.TrimLeft(message, "{")
	message = bytes.TrimSpace(message)

	for {
		index := bytes.Index(message, comma)
		if index == -1 {
			return nil, fmt.Errorf("FrameType was not present in a frame")
		}
		search := bytes.TrimSpace(message[:index])
		if bytes.HasPrefix(search, frameType) {
			typeIndex := bytes.Index(search, semicolon)
			if typeIndex == -1 {
				return nil, fmt.Errorf("problem finding expected value FrameType in frame")
			}
			search = search[typeIndex:]
			return search[2 : len(search)-1], nil // Removes :"" around :"<frameType>"
		}
	}
}
