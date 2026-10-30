package scalers

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
)

type splunkO11yLookbackResponse int

const (
	splunkO11yLookbackData splunkO11yLookbackResponse = iota
	splunkO11yLookbackNoData
	splunkO11yLookbackWaitForStop
)

type splunkO11yLookbackRequest struct {
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	Handle    string `json:"handle"`
	Start     int64  `json:"start"`
	Stop      int64  `json:"stop"`
	Immediate bool   `json:"immediate"`
}

type splunkO11yLookbackLogSink struct {
	processed chan struct{}
}

func (*splunkO11yLookbackLogSink) Init(logr.RuntimeInfo)            {}
func (*splunkO11yLookbackLogSink) Enabled(int) bool                 { return true }
func (*splunkO11yLookbackLogSink) Error(error, string, ...any)      {}
func (l *splunkO11yLookbackLogSink) WithValues(...any) logr.LogSink { return l }
func (l *splunkO11yLookbackLogSink) WithName(string) logr.LogSink   { return l }
func (l *splunkO11yLookbackLogSink) Info(_ int, msg string, _ ...any) {
	if strings.HasPrefix(msg, "Encountering value ") {
		select {
		case l.processed <- struct{}{}:
		default:
		}
	}
}

type splunkO11yLookbackBackend struct {
	response  splunkO11yLookbackResponse
	requests  chan splunkO11yLookbackRequest
	processed chan struct{}
	done      chan struct{}
	server    *httptest.Server
}

func newSplunkO11yLookbackScaler(t *testing.T, response splunkO11yLookbackResponse) (*splunkObservabilityScaler, *splunkO11yLookbackBackend) {
	t.Helper()

	backend := &splunkO11yLookbackBackend{
		response:  response,
		requests:  make(chan splunkO11yLookbackRequest, 1),
		processed: make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	backend.server = httptest.NewServer(http.HandlerFunc(backend.serveHTTP))

	client, err := signalflow.NewClient(signalflow.StreamURL("ws" + strings.TrimPrefix(backend.server.URL, "http")))
	if err != nil {
		backend.server.Close()
		t.Fatal("could not create SignalFlow client:", err)
	}
	scaler := &splunkObservabilityScaler{
		metadata: &splunkObservabilityMetadata{
			Query:            splunkO11yFakeProgram,
			Duration:         30,
			LookbackDuration: 60,
			QueryAggregator:  "max",
		},
		apiClient: client,
		logger:    logr.New(&splunkO11yLookbackLogSink{processed: backend.processed}),
	}
	t.Cleanup(func() {
		close(backend.done)
		_ = scaler.Close(context.Background())
		backend.server.Close()
	})

	return scaler, backend
}

func (b *splunkO11yLookbackBackend) serveHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return len(r.Header.Values("Origin")) == 0
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	channels := make(map[string]string)
	for {
		var request splunkO11yLookbackRequest
		if err := conn.ReadJSON(&request); err != nil {
			return
		}

		switch request.Type {
		case "authenticate":
			if err := conn.WriteJSON(map[string]string{"type": "authenticated"}); err != nil {
				return
			}
		case "execute":
			handle := "handle-" + request.Channel
			channels[handle] = request.Channel
			select {
			case b.requests <- request:
			default:
			}
			if err := conn.WriteJSON(map[string]string{
				"type": "control-message", "channel": request.Channel, "event": "JOB_START", "handle": handle,
			}); err != nil {
				return
			}

			switch b.response {
			case splunkO11yLookbackData:
				if err := b.sendData(conn, request.Channel); err != nil {
					return
				}
				select {
				case <-b.processed:
				case <-b.done:
					return
				}
				if err := b.endChannel(conn, request.Channel, handle); err != nil {
					return
				}
			case splunkO11yLookbackNoData:
				if err := b.endChannel(conn, request.Channel, handle); err != nil {
					return
				}
			case splunkO11yLookbackWaitForStop:
			}
		case "stop":
			if err := b.endChannel(conn, channels[request.Handle], request.Handle); err != nil {
				return
			}
		}
	}
}

func (b *splunkO11yLookbackBackend) sendData(conn *websocket.Conn, channel string) error {
	data := make([]byte, 49)
	data[0], data[1] = 1, 5
	copy(data[4:20], channel)
	binary.BigEndian.PutUint64(data[20:28], 1)
	binary.BigEndian.PutUint32(data[28:32], 1)
	data[32] = byte(messages.ValTypeDouble)
	binary.BigEndian.PutUint64(data[33:41], 1)
	binary.BigEndian.PutUint64(data[41:49], math.Float64bits(42))
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

func (*splunkO11yLookbackBackend) endChannel(conn *websocket.Conn, channel, handle string) error {
	return conn.WriteJSON(map[string]string{
		"type": "control-message", "channel": channel, "event": "END_OF_CHANNEL", "handle": handle,
	})
}

func TestSplunkObservabilityLookbackCompletesFiniteQuery(t *testing.T) {
	scaler, backend := newSplunkO11yLookbackScaler(t, splunkO11yLookbackData)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	value, err := scaler.getQueryResult(ctx)
	if err != nil {
		t.Fatal("expected finite query to succeed:", err)
	}
	if value != 42 {
		t.Fatalf("expected value 42, got %v", value)
	}

	request := <-backend.requests
	if request.Stop-request.Start != int64(60*time.Second/time.Millisecond) {
		t.Fatalf("expected 60 second request window, got start=%d stop=%d", request.Start, request.Stop)
	}
	if !request.Immediate {
		t.Fatal("expected immediate SignalFlow execution")
	}
}

func TestSplunkObservabilityLookbackReturnsNoData(t *testing.T) {
	scaler, _ := newSplunkO11yLookbackScaler(t, splunkO11yLookbackNoData)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := scaler.getQueryResult(ctx)
	if err == nil || !strings.Contains(err.Error(), "query returned no data points") {
		t.Fatalf("expected no-data error, got %v", err)
	}
}

func TestSplunkObservabilityLookbackHonorsContextTimeout(t *testing.T) {
	scaler, _ := newSplunkO11yLookbackScaler(t, splunkO11yLookbackWaitForStop)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := scaler.getQueryResult(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline error, got %v", err)
	}
}
