// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package signalflow

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
	"github.com/signalfx/signalfx-go/idtool"
)

var upgrader = websocket.Upgrader{} // use default options

type tsidVal struct {
	TSID idtool.ID
	Val  float64
}

// FakeBackend is useful for testing, both internal to this package and
// externally.  It supports basic messages and allows for the specification of
// metadata and data messages that map to a particular program.
type FakeBackend struct {
	sync.Mutex

	AccessToken   string
	authenticated bool

	conns map[*websocket.Conn]bool

	received             []map[string]interface{}
	metadataByTSID       map[idtool.ID]*messages.MetadataProperties
	dataByTSID           map[idtool.ID]*float64
	tsidsByProgram       map[string][]idtool.ID
	programErrors        map[string]string
	runningJobsByProgram map[string]int
	cancelFuncsByHandle  map[string]context.CancelFunc
	cancelFuncsByChannel map[string]context.CancelFunc
	server               *httptest.Server
	handleIdx            int

	logger *log.Logger
}

func (f *FakeBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	f.registerConn(c)
	defer c.Close()
	defer cancel()

	textMsgs := make(chan string)
	binMsgs := make(chan []byte)
	go func() {
		for {
			var err error
			select {
			case m := <-textMsgs:
				err = c.WriteMessage(websocket.TextMessage, []byte(m))
			case m := <-binMsgs:
				err = c.WriteMessage(websocket.BinaryMessage, m)
			case <-ctx.Done():
				f.unregisterConn(c)
				return
			}
			if err != nil {
				f.logger.Printf("Could not write message: %v", err)
			}
		}
	}()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				f.logger.Println("read err:", err)
			}
			return
		}

		var in map[string]interface{}
		if err := json.Unmarshal(message, &in); err != nil {
			f.logger.Println("error unmarshalling: ", err)
		}
		f.received = append(f.received, in)

		err = f.handleMessage(ctx, in, textMsgs, binMsgs)
		if err != nil {
			f.logger.Printf("Error handling fake backend message, closing connection: %v", err)
			return
		}
	}
}

func (f *FakeBackend) registerConn(conn *websocket.Conn) {
	f.Lock()
	f.conns[conn] = true
	f.Unlock()
}

func (f *FakeBackend) unregisterConn(conn *websocket.Conn) {
	f.Lock()
	delete(f.conns, conn)
	f.Unlock()
}

func (f *FakeBackend) handleMessage(ctx context.Context, message map[string]interface{}, textMsgs chan<- string, binMsgs chan<- []byte) error {
	typ, ok := message["type"].(string)
	if !ok {
		textMsgs <- `{"type": "error"}`
		return nil
	}

	switch typ {
	case "authenticate":
		token, _ := message["token"].(string)
		if f.AccessToken == "" || token == f.AccessToken {
			textMsgs <- `{"type": "authenticated"}`
			f.authenticated = true
		} else {
			textMsgs <- `{"type": "error", "message": "Invalid auth token"}`
			return errors.New("bad auth token")
		}
	case "stop":
		if cancel := f.cancelFuncsByHandle[message["handle"].(string)]; cancel != nil {
			cancel()
		}
	case "detach":
		if cancel := f.cancelFuncsByChannel[message["channel"].(string)]; cancel != nil {
			cancel()
		}
	case "execute":
		if !f.authenticated {
			return errors.New("not authenticated")
		}
		program, _ := message["program"].(string)
		ch, _ := message["channel"].(string)

		if errMsg := f.programErrors[program]; errMsg != "" {
			textMsgs <- fmt.Sprintf(`{"type": "error", "message": "%s", "channel": "%s"}`, errMsg, ch)
		}

		programTSIDs := f.tsidsByProgram[program]

		handle := fmt.Sprintf("handle-%d", f.handleIdx)
		f.handleIdx++

		execCtx, cancel := context.WithCancel(ctx)
		f.cancelFuncsByHandle[handle] = cancel
		f.cancelFuncsByChannel[ch] = cancel

		f.logger.Printf("Executing SignalFlow program %s with tsids %v and handle %s", program, programTSIDs, handle)
		f.runningJobsByProgram[program]++

		var resolutionMs int
		for _, tsid := range programTSIDs {
			if md := f.metadataByTSID[tsid]; md != nil {
				if md.ResolutionMS > resolutionMs {
					resolutionMs = md.ResolutionMS
				}
			}
		}

		messageResMs, _ := message["resolution"].(float64)
		if messageResMs != 0.0 {
			resolutionMs = int(messageResMs)
		}

		if resolutionMs == 0 {
			resolutionMs = 1000
		}

		// use start and stop to control ending the fakebackend
		var stopMs uint64
		var startMs uint64
		messageStopMs, _ := message["stop"].(float64)
		if messageStopMs != 0.0 {
			stopMs = uint64(messageStopMs)
		}

		messageStartMs, _ := message["start"].(float64)
		if messageStartMs != 0.0 {
			startMs = uint64(messageStartMs)
		}

		if startMs == 0 {
			startMs = uint64(time.Now().UnixNano() / (1000 * 1000))
		}

		textMsgs <- fmt.Sprintf(`{"type": "control-message", "channel": "%s", "event": "STREAM_START"}`, ch)
		textMsgs <- fmt.Sprintf(`{"type": "control-message", "channel": "%s", "event": "JOB_START", "handle": "%s"}`, ch, handle)
		textMsgs <- fmt.Sprintf(`{"type": "message", "channel": "%s", "logicalTimestampMs": 1464736034000, "message": {"contents": {"resolutionMs" : %d}, "messageCode": "JOB_RUNNING_RESOLUTION", "timestampMs": 1464736033000}}`, ch, int64(resolutionMs))

		for _, tsid := range programTSIDs {
			if md := f.metadataByTSID[tsid]; md != nil {
				propJSON, err := json.Marshal(md)
				if err != nil {
					f.logger.Printf("Error serializing metadata to json: %v", err)
					continue
				}
				textMsgs <- fmt.Sprintf(`{"type": "metadata", "tsId": "%s", "channel": "%s", "properties": %s}`, tsid, ch, propJSON)
			}
		}

		f.logger.Print("done sending metadata messages")

		// Send data periodically until the connection is closed.
		iterations := 0
		go func() {
			t := time.NewTicker(time.Duration(resolutionMs) * time.Millisecond)
			for {
				select {
				case <-execCtx.Done():
					f.logger.Printf("sending done")
					f.Lock()
					f.runningJobsByProgram[program]--
					f.Unlock()
					return
				case <-t.C:
					f.Lock()
					valsWithTSID := []tsidVal{}
					for _, tsid := range programTSIDs {
						if data := f.dataByTSID[tsid]; data != nil {
							valsWithTSID = append(valsWithTSID, tsidVal{TSID: tsid, Val: *data})
						}
					}
					f.Unlock()
					metricTime := startMs + uint64(iterations*resolutionMs)
					if stopMs != 0 && metricTime > stopMs {
						f.logger.Printf("sending channel end")
						// tell the client the computation is complete
						textMsgs <- fmt.Sprintf(`{"type": "control-message", "channel": "%s", "event": "END_OF_CHANNEL", "handle": "%s"}`, ch, handle)
						return
					}
					f.logger.Printf("sending data message")
					binMsgs <- makeDataMessage(ch, valsWithTSID, metricTime)
					f.logger.Printf("done sending data message")
					iterations++
				}
			}
		}()
	}
	return nil
}

func makeDataMessage(channel string, valsWithTSID []tsidVal, now uint64) []byte {
	var ch [16]byte
	copy(ch[:], channel)
	header := messages.BinaryMessageHeader{
		Version:     1,
		MessageType: 5,
		Flags:       0,
		Reserved:    0,
		Channel:     ch,
	}
	w := new(bytes.Buffer)
	binary.Write(w, binary.BigEndian, &header)

	dataHeader := messages.DataMessageHeader{
		TimestampMillis: now,
		ElementCount:    uint32(len(valsWithTSID)),
	}
	binary.Write(w, binary.BigEndian, &dataHeader)

	for i := range valsWithTSID {
		var valBytes [8]byte
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, valsWithTSID[i].Val)
		copy(valBytes[:], buf.Bytes())

		payload := messages.DataPayload{
			Type: messages.ValTypeDouble,
			TSID: valsWithTSID[i].TSID,
			Val:  valBytes,
		}

		binary.Write(w, binary.BigEndian, &payload)
	}

	return w.Bytes()
}

func (f *FakeBackend) Start() {
	f.metadataByTSID = map[idtool.ID]*messages.MetadataProperties{}
	f.dataByTSID = map[idtool.ID]*float64{}
	f.tsidsByProgram = map[string][]idtool.ID{}
	f.programErrors = map[string]string{}
	f.runningJobsByProgram = map[string]int{}
	f.cancelFuncsByHandle = map[string]context.CancelFunc{}
	f.cancelFuncsByChannel = map[string]context.CancelFunc{}
	f.conns = map[*websocket.Conn]bool{}
	f.server = httptest.NewServer(f)
}

func (f *FakeBackend) Stop() {
	f.KillExistingConnections()
	f.server.Close()
}

func (f *FakeBackend) Restart() {
	l, err := net.Listen("tcp", f.server.Listener.Addr().String())
	if err != nil {
		panic("Could not relisten: " + err.Error())
	}
	f.server = httptest.NewUnstartedServer(f)
	f.server.Listener = l
	f.server.Start()
}

func (f *FakeBackend) Client() (*Client, error) {
	return NewClient(StreamURL(f.URL()), AccessToken(f.AccessToken))
}

func (f *FakeBackend) AddProgramError(program string, errorMsg string) {
	f.Lock()
	f.programErrors[program] = errorMsg
	f.Unlock()
}

func (f *FakeBackend) AddProgramTSIDs(program string, tsids []idtool.ID) {
	f.Lock()
	f.tsidsByProgram[program] = tsids
	f.Unlock()
}

func (f *FakeBackend) AddTSIDMetadata(tsid idtool.ID, props *messages.MetadataProperties) {
	f.Lock()
	f.metadataByTSID[tsid] = props
	f.Unlock()
}

func (f *FakeBackend) SetTSIDFloatData(tsid idtool.ID, val float64) {
	f.Lock()
	f.dataByTSID[tsid] = &val
	f.Unlock()
}

func (f *FakeBackend) RemoveTSIDData(tsid idtool.ID) {
	f.Lock()
	delete(f.dataByTSID, tsid)
	f.Unlock()
}

func (f *FakeBackend) URL() string {
	return strings.Replace(f.server.URL, "http", "ws", 1)
}

func (f *FakeBackend) KillExistingConnections() {
	f.Lock()
	for conn := range f.conns {
		conn.Close()
	}
	f.Unlock()
}

// RunningJobsForProgram returns how many currently executing jobs there are
// for a particular program text.
func (f *FakeBackend) RunningJobsForProgram(program string) int {
	f.Lock()
	defer f.Unlock()
	return f.runningJobsByProgram[program]
}

// SetLogger sets the internal logger.
func (f *FakeBackend) SetLogger(logger *log.Logger) {
	f.Lock()
	f.logger = logger
	f.Unlock()
}

func NewRunningFakeBackend() *FakeBackend {
	f := &FakeBackend{
		AccessToken: "abcd",
		logger:      log.New(io.Discard, "", 0),
	}
	f.Start()
	return f
}
