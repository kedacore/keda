package scalers

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_getCountFromSeleniumResponse(t *testing.T) {
	type args struct {
		b           []byte
		browserName string
	}
	tests := []struct {
		name    string
		args    args
		want    *resource.Quantity
		wantErr bool
	}{
		{
			name: "nil response body should through error",
			args: args{
				b:           []byte(nil),
				browserName: "",
			},
			// want:    resource.NewQuantity(0, resource.DecimalSI),
			wantErr: true,
		},
		{
			name: "empty response body should through error",
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
						"sessionsInfo": {
							"sessionQueueRequests": [],
							"sessions": []
						}
					}
				}`),
				browserName: "",
			},
			want:    resource.NewQuantity(0, resource.DecimalSI),
			wantErr: false,
		},
		{
			name: "active sessions with no matching browsername should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
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
				browserName: "",
			},
			want:    resource.NewQuantity(0, resource.DecimalSI),
			wantErr: false,
		},
		{
			name: "active sessions with matching browsername should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"sessionsInfo": {
							"sessionQueueRequests": ["{\n  \"browserName\": \"chrome\"\n}","{\n  \"browserName\": \"chrome\"\n}"],
							"sessions": []
						}
					}
				}`),
				browserName: "chrome",
			},
			want:    resource.NewQuantity(2, resource.DecimalSI),
			wantErr: false,
		},
		{
			name: "active sessions with matching browsername should return count as 3",
			args: args{
				b: []byte(`{
					"data": {
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
				browserName: "chrome",
			},
			want:    resource.NewQuantity(3, resource.DecimalSI),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCountFromSeleniumResponse(tt.args.b, tt.args.browserName)
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
		config *ScalerConfig
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
				config: &ScalerConfig{
					TriggerMetadata: map[string]string{},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid browsername string should throw error",
			args: args{
				config: &ScalerConfig{
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
				config: &ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":         "http://selenium-hub:4444/graphql",
						"browserName": "chrome",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				url:         "http://selenium-hub:4444/graphql",
				browserName: "chrome",
				targetValue: 1,
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
