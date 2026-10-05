package scalers

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
)

// The client's Execute registers its computation after sending the request. Wait
// for the scaler's start log before replying so messages cannot race registration.
type splunkO11yLifecycleLogSink struct {
	started   chan struct{}
	processed chan struct{}
}

func (*splunkO11yLifecycleLogSink) Init(logr.RuntimeInfo)            {}
func (*splunkO11yLifecycleLogSink) Enabled(int) bool                 { return true }
func (*splunkO11yLifecycleLogSink) Error(error, string, ...any)      {}
func (l *splunkO11yLifecycleLogSink) WithValues(...any) logr.LogSink { return l }
func (l *splunkO11yLifecycleLogSink) WithName(string) logr.LogSink   { return l }
func (l *splunkO11yLifecycleLogSink) Info(_ int, msg string, _ ...any) {
	if msg == "Started MTS stream." {
		l.started <- struct{}{}
	}
	if strings.HasPrefix(msg, "Encountering value ") {
		l.processed <- struct{}{}
	}
}

type splunkO11yLifecycleConfig struct {
	onStop  bool
	noData  bool
	invalid bool
}

type splunkO11yLifecycleBackend struct {
	splunkO11yLifecycleConfig
	logger *splunkO11yLifecycleLogSink
	ready  chan struct{}
	done   chan struct{}
	server *httptest.Server
}

func newSplunkO11yLifecycleScaler(t *testing.T, duration int, config splunkO11yLifecycleConfig) (*splunkObservabilityScaler, *splunkO11yLifecycleBackend) {
	t.Helper()
	b := &splunkO11yLifecycleBackend{
		logger: &splunkO11yLifecycleLogSink{
			started:   make(chan struct{}, 1),
			processed: make(chan struct{}, 1),
		},
		splunkO11yLifecycleConfig: config,
		ready:                     make(chan struct{}, 1), done: make(chan struct{}),
	}
	b.server = httptest.NewServer(http.HandlerFunc(b.serveHTTP))
	client, err := signalflow.NewClient(signalflow.StreamURL("ws" + strings.TrimPrefix(b.server.URL, "http")))
	if err != nil {
		b.server.Close()
		t.Fatal(err)
	}
	scaler := &splunkObservabilityScaler{
		metadata: &splunkObservabilityMetadata{
			Query: splunkO11yFakeProgram, Duration: duration, QueryAggregator: "max",
		},
		apiClient: client, logger: logr.New(b.logger),
	}
	t.Cleanup(func() {
		close(b.done)
		_ = scaler.Close(context.Background())
		b.server.Close()
	})
	return scaler, b
}

func (b *splunkO11yLifecycleBackend) serveHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	channels := make(map[string]string)
	for {
		var request struct {
			Type    string `json:"type"`
			Channel string `json:"channel"`
			Handle  string `json:"handle"`
		}
		if err := conn.ReadJSON(&request); err != nil {
			return
		}
		switch request.Type {
		case "authenticate":
			if err := conn.WriteJSON(map[string]string{"type": "authenticated"}); err != nil {
				return
			}
		case "execute":
			select {
			case <-b.logger.started:
			case <-b.done:
				return
			}
			handle := "handle-" + request.Channel
			channels[handle] = request.Channel
			if err := conn.WriteJSON(map[string]string{"type": "control-message", "channel": request.Channel, "event": "JOB_START", "handle": handle}); err != nil {
				return
			}
			select {
			case b.ready <- struct{}{}:
			default:
			}
			if b.onStop {
				continue
			}
			if b.invalid {
				if err := b.sendData(conn, request.Channel); err != nil {
					return
				}
				continue // The processing error must cause a Stop and drain the remaining outputs.
			}
			if err := b.complete(conn, request.Channel, handle); err != nil {
				return
			}
		case "stop":
			if err := b.complete(conn, channels[request.Handle], request.Handle); err != nil {
				return
			}
		}
	}
}

func (b *splunkO11yLifecycleBackend) complete(conn *websocket.Conn, channel, handle string) error {
	auxiliary := []map[string]any{
		{"type": "message", "channel": channel, "message": map[string]any{
			"messageCode": "JOB_RUNNING_RESOLUTION", "contents": map[string]int{"resolutionMs": 1000},
		}},
		{"type": "event", "channel": channel},
		{"type": "expired-tsid", "channel": channel, "tsId": "AAAAAAAAAAE"},
	}
	for _, msg := range auxiliary {
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}
	if !b.noData && !b.invalid {
		if err := b.sendData(conn, channel); err != nil {
			return err
		}
		// Text END and binary Data use separate client queues. Wait for Data to
		// be consumed rather than relying on their relative scheduling order.
		select {
		case <-b.logger.processed:
		case <-b.done:
			return context.Canceled
		}
	}
	return conn.WriteJSON(map[string]string{"type": "control-message", "channel": channel, "event": "END_OF_CHANNEL", "handle": handle})
}

func (b *splunkO11yLifecycleBackend) sendData(conn *websocket.Conn, channel string) error {
	data := make([]byte, 49) // 20-byte message header, 12-byte data header, 17-byte payload.
	data[0], data[1] = 1, 5
	copy(data[4:20], channel)
	binary.BigEndian.PutUint64(data[20:28], 1)
	binary.BigEndian.PutUint32(data[28:32], 1)
	data[32] = byte(messages.ValTypeDouble)
	if b.invalid {
		data[32] = byte(messages.ValTypeLong)
	}
	binary.BigEndian.PutUint64(data[33:41], 1)
	binary.BigEndian.PutUint64(data[41:49], math.Float64bits(42))
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

func countSplunkO11yBufferGoroutines() int {
	buf := make([]byte, 2<<20)
	n := runtime.Stack(buf, true)
	count := 0
	for _, stack := range strings.Split(string(buf[:n]), "\n\n") {
		if strings.Contains(stack, "signalflow-client-go") && strings.Contains(stack, "bufferMessages") {
			count++
		}
	}
	return count
}

func assertSplunkO11yBuffersReaped(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		got := countSplunkO11yBufferGoroutines()
		if got <= baseline {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("retained %d SignalFlow output goroutines (baseline %d)", got-baseline, baseline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSplunkObservabilityRepeatedQueriesDrainAllOutputs(t *testing.T) {
	for _, noData := range []bool{false, true} {
		t.Run(fmt.Sprintf("noData=%v", noData), func(t *testing.T) {
			baseline := countSplunkO11yBufferGoroutines()
			scaler, _ := newSplunkO11yLifecycleScaler(t, 3600, splunkO11yLifecycleConfig{noData: noData})
			for i := 0; i < 100; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				value, err := scaler.getQueryResult(ctx)
				cancel()
				if noData {
					if err == nil || !strings.Contains(err.Error(), "query returned no data points") {
						t.Fatalf("query %d: expected no-data error, got %v", i, err)
					}
				} else if err != nil || value != 42 {
					t.Fatalf("query %d: got value %v, error %v", i, value, err)
				}
				assertSplunkO11yBuffersReaped(t, baseline)
			}
		})
	}
}

func TestSplunkObservabilityStopDrainsAllOutputs(t *testing.T) {
	baseline := countSplunkO11yBufferGoroutines()
	scaler, _ := newSplunkO11yLifecycleScaler(t, 0, splunkO11yLifecycleConfig{onStop: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	value, err := scaler.getQueryResult(ctx)
	if err != nil || value != 42 {
		t.Fatalf("stopped query: got value %v, error %v", value, err)
	}
	assertSplunkO11yBuffersReaped(t, baseline)
}

func TestSplunkObservabilityProcessingErrorDrainsAllOutputs(t *testing.T) {
	baseline := countSplunkO11yBufferGoroutines()
	scaler, _ := newSplunkO11yLifecycleScaler(t, 3600, splunkO11yLifecycleConfig{invalid: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := scaler.getQueryResult(ctx)
	if err == nil || !strings.Contains(err.Error(), "could not convert") {
		t.Fatalf("expected metric processing error, got %v", err)
	}
	assertSplunkO11yBuffersReaped(t, baseline)
}

func TestSplunkObservabilityStopDrainsAfterProcessingError(t *testing.T) {
	baseline := countSplunkO11yBufferGoroutines()
	scaler, backend := newSplunkO11yLifecycleScaler(t, 0, splunkO11yLifecycleConfig{onStop: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	comp, err := scaler.apiClient.Execute(ctx, &signalflow.ExecuteRequest{Program: splunkO11yFakeProgram})
	if err != nil {
		t.Fatal(err)
	}
	backend.logger.started <- struct{}{}
	processErr := errors.New("cannot process metric")
	err = scaler.stopAndDrain(comp, func(*messages.DataMessage) error {
		backend.logger.processed <- struct{}{}
		return processErr
	})
	if !errors.Is(err, processErr) {
		t.Fatalf("expected processing error after draining, got %v", err)
	}
	assertSplunkO11yBuffersReaped(t, baseline)
}

func TestSplunkObservabilityCancellationDrainsAllOutputs(t *testing.T) {
	for _, closeScaler := range []bool{false, true} {
		t.Run(fmt.Sprintf("close=%v", closeScaler), func(t *testing.T) {
			baseline := countSplunkO11yBufferGoroutines()
			scaler, backend := newSplunkO11yLifecycleScaler(t, 3600, splunkO11yLifecycleConfig{onStop: true, noData: true})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := scaler.getQueryResult(ctx)
				result <- err
			}()
			select {
			case <-backend.ready:
			case <-time.After(5 * time.Second):
				t.Fatal("query did not start")
			}
			closeDone := make(chan error, 1)
			if closeScaler {
				go func() { closeDone <- scaler.Close(context.Background()) }()
			} else {
				cancel()
			}
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected cancellation error, got %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("query cancellation waited for the cleanup deadline")
			}
			if closeScaler {
				select {
				case err := <-closeDone:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(splunkO11yDrainTimeout + time.Second):
					t.Fatal("Close did not complete")
				}
			}
			assertSplunkO11yBuffersReaped(t, baseline)
		})
	}
}
