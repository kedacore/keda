package scalers

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func Test_getCountFromSeleniumResponse(t *testing.T) {
	type args struct {
		b                     []byte
		browserName           string
		sessionBrowserName    string
		browserVersion        string
		platformName          string
		sessionsPerNode       int64
		setSessionsFromHub    bool
		sessionBrowserVersion string
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "nil response body should throw error",
			args: args{
				b:           []byte(nil),
				browserName: "",
			},
			// want:    0,
			wantErr: true,
		},
		{
			name: "empty response body should throw error",
			args: args{
				b:           []byte(""),
				browserName: "",
			},
			// want:    resource.NewQuantity(0, resource.DecimalSI),
			wantErr: true,
		},
		{
			name: "no active sessions should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 0,
							"nodeCount": 0
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": [],
							"sessions": []
						}
					}
				}`),
				browserName: "",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "active sessions with no matching browsername should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								}
							]
						}
					}
				}`),
				browserName:        "",
				sessionBrowserName: "",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "active sessions with matching browsername should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": []
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "2 session queue with matching browsername and browserversion should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 4,
							"nodeCount": 2
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}"]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "2 active sessions with matching browsername on 2 nodes and maxSession=4 should return count as 1 (rounded up from 0.75)",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 4,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								},
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b2",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983d"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "2 active sessions with matching browsername on 1 node and maxSession=3 should return count as 1 (rounded up from 0.33)",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 3,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								},
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b2",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983d"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "2 active sessions with matching browsername on 2 nodes should return count as 5",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 2,
							"nodeCount": 2
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								},
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b2",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983d"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "2 active sessions with matching browsername on 2 nodes with 3 other versions in queue should return count as 2 with default browserVersion and PlatformName",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 2,
							"nodeCount": 2
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								},
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b2",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983d"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "2 active sessions with matching browsername on 2 nodes should return count as 5 with default browserVersion / PlatformName and incoming sessions do not have versions",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 2,
							"nodeCount": 2
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\"}","{\n  \"browserName\": \"chrome\"}","{\n  \"browserName\": \"chrome\"}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								},
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b2",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983d"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "1 active session with matching browsername and version should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "1 active msedge session with matching browsername/sessionBrowserName should return count as 3",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"MicrosoftEdge\",\n \"browserVersion\": \"91.0\"\n}","{\n  \"browserName\": \"MicrosoftEdge\",\n \"browserVersion\": \"91.0\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"msedge\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"msedge\": {\n    \"msedgedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"ms:edgeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								}
							]
						}
					}
				}`),
				browserName:        "MicrosoftEdge",
				sessionBrowserName: "msedge",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "1 active msedge session while asking for 2 chrome sessions should return a count of 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"msedge\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"msedge\": {\n    \"msedgedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"ms:edgeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "1 active msedge session with maxSessions=3 while asking for 3 chrome sessions should return a count of 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 3,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"msedge\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"msedge\": {\n    \"msedgedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"ms:edgeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "session request with matching browsername and no specific platformName should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"],
							"sessions": []
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "Windows 11",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "sessions requests with matching browsername and platformName should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"],
							"sessions": []
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "Windows 11",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "sessions requests with matching browsername and platformName when setSessionsFromHub turned on and node with 2 slots matches should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"stereotypes":"[{\"slots\":1,\"stereotype\":{\"browserName\":\"chrome\",\"platformName\":\"linux\"}}]"
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"],
							"sessions": []
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
				setSessionsFromHub: true,
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "4 sessions requests with matching browsername and platformName when setSessionsFromHub turned on and node with 2 slots matches should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"stereotypes":"[{\"slots\":2,\"stereotype\":{\"browserName\":\"chrome\",\"platformName\":\"linux\"}}]"
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"],
							"sessions": []
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
				setSessionsFromHub: true,
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "4 sessions requests with matching browsername and platformName when setSessionsFromHub turned on, no nodes and sessionsPerNode=2 matches should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"],
							"sessions": []
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
				setSessionsFromHub: true,
				sessionsPerNode:    2,
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "sessions requests and active sessions with matching browsername and platformName should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\",\n \"browserVersion\": \"91.0\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"Windows 11\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								},
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								}
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "Windows 11",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "sessions requests and active sessions with matching browsername, platformName and sessionBrowserVersion should return count as 3",
			args: args{
				b: []byte(`{
					"data": {
						"grid":{
							"maxSession": 1,
							"nodeCount": 1
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}","{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\",\n \"browserVersion\": \"91.0\"\n}"],
							"sessions": [
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								},
								{
									"id": "0f9c5a941aa4d755a54b84be1f6535b1",
									"capabilities": "{\n  \"acceptInsecureCerts\": false,\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"91.0.4472.114\",\n  \"chrome\": {\n    \"chromedriverVersion\": \"91.0.4472.101 (af52a90bf87030dd1523486a1cd3ae25c5d76c9b-refs\\u002fbranch-heads\\u002f4472@{#1462})\",\n    \"userDataDir\": \"\\u002ftmp\\u002f.com.google.Chrome.DMqx9m\"\n  },\n  \"goog:chromeOptions\": {\n    \"debuggerAddress\": \"localhost:35839\"\n  },\n  \"networkConnectionEnabled\": false,\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"proxy\": {\n  },\n  \"se:cdp\": \"http:\\u002f\\u002flocalhost:35839\",\n  \"se:cdpVersion\": \"91.0.4472.114\",\n  \"se:vncEnabled\": true,\n  \"se:vncLocalAddress\": \"ws:\\u002f\\u002flocalhost:7900\\u002fwebsockify\",\n  \"setWindowRect\": true,\n  \"strictFileInteractability\": false,\n  \"timeouts\": {\n    \"implicit\": 0,\n    \"pageLoad\": 300000,\n    \"script\": 30000\n  },\n  \"unhandledPromptBehavior\": \"dismiss and notify\",\n  \"webauthn:extension:largeBlob\": true,\n  \"webauthn:virtualAuthenticators\": true\n}",
									"nodeId": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c"
								}
							]
						}
					}
				}`),
				browserName:           "chrome",
				sessionBrowserName:    "chrome",
				sessionBrowserVersion: "91.0.4472.114",
				platformName:          "linux",
			},
			want:    3,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCountFromSeleniumResponse(tt.args.b, tt.args.browserName, tt.args.browserVersion, tt.args.sessionBrowserName, tt.args.platformName, tt.args.sessionsPerNode, tt.args.setSessionsFromHub, tt.args.sessionBrowserVersion, logr.Discard())
			if (err != nil) != tt.wantErr {
				t.Errorf("getCountFromSeleniumResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCountFromSeleniumResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseSeleniumGridScalerMetadata(t *testing.T) {
	type args struct {
		config *scalersconfig.ScalerConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *seleniumGridScalerMetadata
		wantErr bool
	}{
		{
			name: "invalid url string should throw error",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid browsername string should throw error",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url": "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid url and browsername should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":         "http://selenium-hub:4444/graphql",
						"browserName": "chrome",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                "http://selenium-hub:4444/graphql",
				BrowserName:        "chrome",
				SessionBrowserName: "chrome",
				TargetValue:        1,
				BrowserVersion:     "latest",
				PlatformName:       "linux",
				SessionsPerNode:    1,
			},
		},
		{
			name: "valid url, browsername, and sessionbrowsername should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                "http://selenium-hub:4444/graphql",
						"browserName":        "MicrosoftEdge",
						"sessionBrowserName": "msedge",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                "http://selenium-hub:4444/graphql",
				BrowserName:        "MicrosoftEdge",
				SessionBrowserName: "msedge",
				TargetValue:        1,
				BrowserVersion:     "latest",
				PlatformName:       "linux",
				SessionsPerNode:    1,
			},
		},
		{
			name: "valid url in AuthParams, browsername, and sessionbrowsername should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					AuthParams: map[string]string{
						"url": "http://user:password@selenium-hub:4444/graphql",
					},
					TriggerMetadata: map[string]string{
						"browserName":        "MicrosoftEdge",
						"sessionBrowserName": "msedge",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                "http://user:password@selenium-hub:4444/graphql",
				BrowserName:        "MicrosoftEdge",
				SessionBrowserName: "msedge",
				TargetValue:        1,
				BrowserVersion:     "latest",
				PlatformName:       "linux",
				SessionsPerNode:    1,
			},
		},
		{
			name: "valid url and browsername should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":            "http://selenium-hub:4444/graphql",
						"browserName":    "chrome",
						"browserVersion": "91.0",
						"unsafeSsl":      "false",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                "http://selenium-hub:4444/graphql",
				BrowserName:        "chrome",
				SessionBrowserName: "chrome",
				TargetValue:        1,
				BrowserVersion:     "91.0",
				UnsafeSsl:          false,
				PlatformName:       "linux",
				SessionsPerNode:    1,
			},
		},
		{
			name: "valid url, browsername, unsafeSsl and activationThreshold should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                 "http://selenium-hub:4444/graphql",
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "10",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                 "http://selenium-hub:4444/graphql",
				BrowserName:         "chrome",
				SessionBrowserName:  "chrome",
				TargetValue:         1,
				ActivationThreshold: 10,
				BrowserVersion:      "91.0",
				UnsafeSsl:           true,
				PlatformName:        "linux",
				SessionsPerNode:     1,
			},
		},
		{
			name: "valid url, browsername and unsafeSsl but invalid activationThreshold should throw an error",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                 "http://selenium-hub:4444/graphql",
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "AA",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid url, browsername, unsafeSsl and activationThreshold with default platformName should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                 "http://selenium-hub:4444/graphql",
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "10",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                 "http://selenium-hub:4444/graphql",
				BrowserName:         "chrome",
				SessionBrowserName:  "chrome",
				TargetValue:         1,
				ActivationThreshold: 10,
				BrowserVersion:      "91.0",
				UnsafeSsl:           true,
				PlatformName:        "linux",
				SessionsPerNode:     1,
			},
		},
		{
			name: "valid url, browsername, unsafeSsl, activationThreshold and platformName should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                 "http://selenium-hub:4444/graphql",
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "10",
						"platformName":        "Windows 11",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                 "http://selenium-hub:4444/graphql",
				BrowserName:         "chrome",
				SessionBrowserName:  "chrome",
				TargetValue:         1,
				ActivationThreshold: 10,
				BrowserVersion:      "91.0",
				UnsafeSsl:           true,
				PlatformName:        "Windows 11",
				SessionsPerNode:     1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSeleniumGridScalerMetadata(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSeleniumGridScalerMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSeleniumGridScalerMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
