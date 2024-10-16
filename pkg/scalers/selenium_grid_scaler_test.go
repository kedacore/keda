package scalers

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func Test_getCountFromSeleniumResponse(t *testing.T) {
	type args struct {
		b                  []byte
		browserName        string
		sessionBrowserName string
		browserVersion     string
		platformName       string
		nodeMaxSessions    int
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
			wantErr: true,
		},
		{
			name: "empty response body should throw error",
			args: args{
				b:           []byte(""),
				browserName: "",
			},
			wantErr: true,
		},
		{
			name: "no sessionQueueRequests should return count as 0",
			args: args{
				b: []byte(`{
					  "data": {
						"grid": {
						  "sessionCount": 0,
						  "maxSession": 0,
						  "totalSlots": 0
						},
						"nodesInfo": {
						  "nodes": []
						},
						"sessionsInfo": {
						  "sessionQueueRequests": []
						}
					  }
					}
				`),
				browserName: "",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "12 sessionQueueRequests with 4 requests matching browserName chrome should return count as 4",
			args: args{
				b: []byte(`{
				  "data": {
					"grid": {
					  "sessionCount": 0,
					  "maxSession": 0,
					  "totalSlots": 0
					},
					"nodesInfo": {
					  "nodes": []
					},
					"sessionsInfo": {
					  "sessionQueueRequests": [
						"{\n  \"browserName\": \"chrome\",\n  \"goog:chromeOptions\": {\n    \"extensions\": [\n    ],\n    \"args\": [\n      \"disable-features=DownloadBubble,DownloadBubbleV2\"\n    ]\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_download_file (ChromeTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_with_frames (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_download_file (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_title_and_maximize_window (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"browserName\": \"chrome\",\n  \"goog:chromeOptions\": {\n    \"extensions\": [\n    ],\n    \"args\": [\n      \"disable-features=DownloadBubble,DownloadBubbleV2\"\n    ]\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_play_video (ChromeTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"browserName\": \"chrome\",\n  \"goog:chromeOptions\": {\n    \"extensions\": [\n    ],\n    \"args\": [\n      \"disable-features=DownloadBubble,DownloadBubbleV2\"\n    ]\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_select_from_a_dropdown (ChromeTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_visit_basic_auth_secured_page (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_select_from_a_dropdown (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"browserName\": \"chrome\",\n  \"goog:chromeOptions\": {\n    \"extensions\": [\n    ],\n    \"args\": [\n      \"disable-features=DownloadBubble,DownloadBubbleV2\"\n    ]\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_title (ChromeTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_title (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_accept_languages (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_play_video (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}"
					  ]
					}
				  }
				}
				`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    4,
			wantErr: false,
		},
		{
			name: "2 sessionQueueRequests and 1 available nodeStereotypes with matching browserName firefox should return count as 1",
			args: args{
				b: []byte(`{
					  "data": {
						"grid": {
						  "sessionCount": 0,
						  "maxSession": 7,
						  "totalSlots": 7
						},
						"nodesInfo": {
						  "nodes": [
							{
							  "id": "82ee33bd-390e-4dd6-aee2-06b17ecee18e",
							  "status": "UP",
							  "sessionCount": 1,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"chrome\",\n      \"browserVersion\": \"128.0\",\n      \"goog:chromeOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-chrome-name-m5n8z-4br6x\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": [
								{
								  "id": "reserved",
								  "capabilities": "{\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"128.0\",\n  \"goog:chromeOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-chrome-name-m5n8z-4br6x\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}",
								  "slot": {
									"id": "83c9d9f5-f79d-4dea-bc9b-ce61bf2bc01c",
									"stereotype": "{\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"128.0\",\n  \"goog:chromeOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-chrome-name-m5n8z-4br6x\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}"
								  }
								}
							  ]
							},
							{
							  "id": "b4d3d31a-3239-4c09-a5f5-3650d4fcef48",
							  "status": "UP",
							  "sessionCount": 1,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"firefox\",\n      \"browserVersion\": \"130.0\",\n      \"moz:firefoxOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-firefox-name-s2gq6-82lwb\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": [
								{
								  "id": "reserved",
								  "capabilities": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-s2gq6-82lwb\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}",
								  "slot": {
									"id": "b03b80c0-95f8-4b9c-ba06-bebd2568ce3d",
									"stereotype": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-s2gq6-82lwb\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}"
								  }
								}
							  ]
							},
							{
							  "id": "f3e67bf7-3c40-42d4-ab10-666b49c88925",
							  "status": "UP",
							  "sessionCount": 0,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"chrome\",\n      \"browserVersion\": \"128.0\",\n      \"goog:chromeOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-chrome-name-xh95p-9c2cl\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": []
							},
							{
							  "id": "f1e315fe-5f32-4a73-bb31-b73ed9a728e5",
							  "status": "UP",
							  "sessionCount": 1,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"chrome\",\n      \"browserVersion\": \"128.0\",\n      \"goog:chromeOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-chrome-name-j2xbn-lq76c\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": [
								{
								  "id": "reserved",
								  "capabilities": "{\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"128.0\",\n  \"goog:chromeOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-chrome-name-j2xbn-lq76c\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}",
								  "slot": {
									"id": "9d91cd87-b443-4a0c-93e7-eea8c4661207",
									"stereotype": "{\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"128.0\",\n  \"goog:chromeOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-chrome-name-j2xbn-lq76c\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}"
								  }
								}
							  ]
							},
							{
							  "id": "0ae48415-a230-4bc4-a26c-4fc4ffc3abc1",
							  "status": "UP",
							  "sessionCount": 1,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"firefox\",\n      \"browserVersion\": \"130.0\",\n      \"moz:firefoxOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-firefox-name-xk6mm-2m6jh\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": [
								{
								  "id": "reserved",
								  "capabilities": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-xk6mm-2m6jh\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}",
								  "slot": {
									"id": "2c1fc5c4-881a-48fd-9b9e-b4d3ecbc1bd8",
									"stereotype": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-xk6mm-2m6jh\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}"
								  }
								}
							  ]
							},
							{
							  "id": "284fa982-5be0-44a6-b64e-e2e76fe52d1f",
							  "status": "UP",
							  "sessionCount": 1,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"firefox\",\n      \"browserVersion\": \"130.0\",\n      \"moz:firefoxOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-firefox-name-bvq59-6dh6q\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": [
								{
								  "id": "reserved",
								  "capabilities": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-bvq59-6dh6q\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}",
								  "slot": {
									"id": "5f8f9ba0-0f61-473e-b367-b68d9368dc24",
									"stereotype": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-bvq59-6dh6q\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}"
								  }
								}
							  ]
							},
							{
							  "id": "451442d0-3649-4b21-a5a5-32bc847f1765",
							  "status": "UP",
							  "sessionCount": 0,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"firefox\",\n      \"browserVersion\": \"130.0\",\n      \"moz:firefoxOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-firefox-name-42xbf-zpdd4\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": []
							},
							{
							  "id": "a4d26330-e5be-4630-b4da-9078f2495ece",
							  "status": "UP",
							  "sessionCount": 1,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"firefox\",\n      \"browserVersion\": \"130.0\",\n      \"moz:firefoxOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-firefox-name-qt9z2-6xx86\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": [
								{
								  "id": "reserved",
								  "capabilities": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-qt9z2-6xx86\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}",
								  "slot": {
									"id": "38bd0b09-ffe0-46e9-8983-bd208270c8da",
									"stereotype": "{\n  \"browserName\": \"firefox\",\n  \"browserVersion\": \"130.0\",\n  \"moz:firefoxOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-firefox-name-qt9z2-6xx86\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}"
								  }
								}
							  ]
							},
							{
							  "id": "e81f0038-fc72-4045-9de1-b98143053eae",
							  "status": "UP",
							  "sessionCount": 1,
							  "maxSession": 1,
							  "slotCount": 1,
							  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"chrome\",\n      \"browserVersion\": \"128.0\",\n      \"goog:chromeOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-chrome-name-v7nrv-xsfkb\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
							  "sessions": [
								{
								  "id": "reserved",
								  "capabilities": "{\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"128.0\",\n  \"goog:chromeOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-chrome-name-v7nrv-xsfkb\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}",
								  "slot": {
									"id": "43b992cc-39bb-4b0f-92b6-99603a543459",
									"stereotype": "{\n  \"browserName\": \"chrome\",\n  \"browserVersion\": \"128.0\",\n  \"goog:chromeOptions\": {\n    \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n  },\n  \"platformName\": \"linux\",\n  \"se:containerName\": \"my-chrome-name-v7nrv-xsfkb\",\n  \"se:downloadsEnabled\": true,\n  \"se:noVncPort\": 7900,\n  \"se:vncEnabled\": true\n}"
								  }
								}
							  ]
							}
						  ]
						},
						"sessionsInfo": {
						  "sessionQueueRequests": [
							"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_accept_languages (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
							"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_play_video (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}"
						  ]
						}
					  }
					}
				`),
				browserName:        "firefox",
				sessionBrowserName: "firefox",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "1 sessionQueueRequests and 1 available nodeStereotypes with matching browserName chrome should return count as 0",
			args: args{
				b: []byte(`{
				  "data": {
					"grid": {
					  "sessionCount": 0,
					  "maxSession": 0,
					  "totalSlots": 0
					},
					"nodesInfo": {
					  "nodes": [
						{
						  "id": "f3e67bf7-3c40-42d4-ab10-666b49c88925",
						  "status": "UP",
						  "sessionCount": 0,
						  "maxSession": 1,
						  "slotCount": 1,
						  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"chrome\",\n      \"browserVersion\": \"128.0\",\n      \"goog:chromeOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-chrome-name-xh95p-9c2cl\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
						  "sessions": []
						},
						{
						  "id": "451442d0-3649-4b21-a5a5-32bc847f1765",
						  "status": "UP",
						  "sessionCount": 0,
						  "maxSession": 1,
						  "slotCount": 1,
						  "stereotypes": "[\n  {\n    \"slots\": 1,\n    \"stereotype\": {\n      \"browserName\": \"firefox\",\n      \"browserVersion\": \"130.0\",\n      \"moz:firefoxOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002ffirefox\"\n      },\n      \"platformName\": \"linux\",\n      \"se:containerName\": \"my-firefox-name-42xbf-zpdd4\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
						  "sessions": []
						}
					  ]
					},
					"sessionsInfo": {
					  "sessionQueueRequests": [
						"{\n  \"acceptInsecureCerts\": true,\n  \"browserName\": \"firefox\",\n  \"moz:debuggerAddress\": true,\n  \"moz:firefoxOptions\": {\n    \"prefs\": {\n      \"remote.active-protocols\": 3\n    },\n    \"profile\": \"profile\"\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_accept_languages (FirefoxTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}",
						"{\n  \"browserName\": \"chrome\",\n  \"goog:chromeOptions\": {\n    \"extensions\": [\n    ],\n    \"args\": [\n      \"disable-features=DownloadBubble,DownloadBubbleV2\"\n    ]\n  },\n  \"pageLoadStrategy\": \"normal\",\n  \"platformName\": \"linux\",\n  \"se:downloadsEnabled\": true,\n  \"se:name\": \"test_visit_basic_auth_secured_page (ChromeTests)\",\n  \"se:recordVideo\": true,\n  \"se:screenResolution\": \"1920x1080\"\n}"
					  ]
					}
				  }
				}
				`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "1 sessionQueueRequests Linux and 1 available nodeStereotypes Windows with matching browserName chrome should return count as 1",
			args: args{
				b: []byte(`{
				  "data": {
					"grid": {
					  "sessionCount": 0,
					  "maxSession": 2,
					  "totalSlots": 2
					},
					"nodesInfo": {
					  "nodes": [
						{
						  "id": "node-1",
						  "status": "UP",
						  "sessionCount": 0,
						  "maxSession": 1,
						  "slotCount": 1,
						  "stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Windows 11\"}}]",
						  "sessions": []
						},
						{
						  "id": "node-2",
						  "status": "UP",
						  "sessionCount": 0,
						  "maxSession": 1,
						  "slotCount": 1,
						  "stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"firefox\", \"browserVersion\": \"130.0\", \"platformName\": \"Windows 11\"}}]",
						  "sessions": []
						}
					  ]
					},
					"sessionsInfo": {
					  "sessionQueueRequests": [
						"{\"browserName\": \"firefox\", \"browserVersion\": \"130.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}"
					  ]
					}
				  }
				}
				`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "scaler browserVersion is latest, 2 sessionQueueRequests wihtout browserVersion, 2 available nodeStereotypes with different versions and platforms, should return count as 1",
			args: args{
				b: []byte(`{
                    "data": {
                        "grid": {
                            "sessionCount": 0,
                            "maxSession": 0,
                            "totalSlots": 0
                        },
                        "nodesInfo": {
                            "nodes": [
                                {
                                    "id": "node-1",
                                    "status": "UP",
                                    "sessionCount": 0,
                                    "maxSession": 1,
                                    "slotCount": 1,
                                    "stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
                                    "sessions": []
                                },
                                {
                                    "id": "node-2",
                                    "status": "UP",
                                    "sessionCount": 0,
                                    "maxSession": 1,
                                    "slotCount": 1,
                                    "stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"92.0\", \"platformName\": \"Windows 11\"}}]",
                                    "sessions": []
                                }
                            ]
                        },
                        "sessionsInfo": {
                            "sessionQueueRequests": [
                                "{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
                                "{\"browserName\": \"chrome\", \"platformName\": \"linux\"}"
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
			name: "scaler browserVersion is latest, 5 sessionQueueRequests wihtout browserVersion also 1 different platformName, 1 available nodeStereotypes with 3 slots Linux and 1 node Windows, should return count as 1",
			args: args{
				b: []byte(`{
                    "data": {
                        "grid": {
                            "sessionCount": 0,
                            "maxSession": 6,
                            "totalSlots": 6
                        },
                        "nodesInfo": {
                            "nodes": [
                                {
                                    "id": "node-1",
                                    "status": "UP",
                                    "sessionCount": 0,
                                    "maxSession": 3,
                                    "slotCount": 3,
                                    "stereotypes": "[{\"slots\": 3, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
                                    "sessions": []
                                },
                                {
                                    "id": "node-2",
                                    "status": "UP",
                                    "sessionCount": 0,
                                    "maxSession": 3,
                                    "slotCount": 3,
                                    "stereotypes": "[{\"slots\": 3, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"92.0\", \"platformName\": \"Windows 11\"}}]",
                                    "sessions": []
                                }
                            ]
                        },
                        "sessionsInfo": {
                            "sessionQueueRequests": [
                                "{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
                                "{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
                                "{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
                                "{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
                                "{\"browserName\": \"chrome\", \"platformName\": \"Windows 11\"}"
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
			name: "queue request with browserName browserVersion and browserVersion but no available nodes should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"firefox\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"firefox\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
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
			name: "1 queue request with browserName browserVersion and browserVersion but 2 nodes without available slots should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								},
								{
									"id": "node-2",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-2",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
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
			name: "2 session queue with matching browsername and browserversion of 2 available slots should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 0,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 0,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": []
								},
								{
									"id": "node-2",
									"status": "UP",
									"sessionCount": 0,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": []
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "2 queue requests with browserName browserVersion and platformName matching 2 available slots on 2 different nodes should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 4,
							"totalSlots": 4
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 2,
									"slotCount": 2,
									"stereotypes": "[{\"slots\": 2, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								},
								{
									"id": "node-2",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 2,
									"slotCount": 2,
									"stereotypes": "[{\"slots\": 2, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-2",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "1 queue request with browserName browserVersion and platformName matching 1 available slot on node has 3 max sessions should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 3,
							"totalSlots": 3
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 2,
									"maxSession": 3,
									"slotCount": 3,
									"stereotypes": "[{\"slots\": 3, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										},
										{
											"id": "session-2",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "3 queue requests with browserName browserVersion and platformName but 2 running nodes are busy should return count as 3",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								},
								{
									"id": "node-2",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-2",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "3 queue requests with browserName browserVersion and platformName but 2 running nodes are busy with different versions should return count as 3",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								},
								{
									"id": "node-2",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-2",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"90.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"92.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"93.0\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "3 queue requests with browserName and platformName but 2 running nodes are busy with different versions should return count as 3",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								},
								{
									"id": "node-2",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-2",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "1 active session with matching browsername and version should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
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
			name: "1 request without browserName and browserVersion stable can be match any available node should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 0,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 0,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"v128.0\", \"platformName\": \"linux\"}}]",
									"sessions": []
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserVersion\": \"stable\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "1 request without browserName and browserVersion stable should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"v128.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"v128.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserVersion\": \"stable\", \"platformName\": \"linux\"}"
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
			name: "2 queue requests with browserName in string match node stereotype and scaler metadata browserVersion should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"msedge\", \"browserVersion\": \"dev\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"msedge\", \"browserVersion\": \"dev\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"MicrosoftEdge\", \"browserVersion\": \"beta\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"MicrosoftEdge\", \"browserVersion\": \"dev\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "MicrosoftEdge",
				sessionBrowserName: "msedge",
				browserVersion:     "dev",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "2 queue requests with matching browsername/sessionBrowserName but 1 node is busy should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"msedge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"msedge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"MicrosoftEdge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"MicrosoftEdge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "MicrosoftEdge",
				sessionBrowserName: "msedge",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "2 queue requests with matching browsername/sessionBrowserName and 1 node is is available should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 0,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 0,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"msedge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": []
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"MicrosoftEdge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"MicrosoftEdge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "MicrosoftEdge",
				sessionBrowserName: "msedge",
				browserVersion:     "91.0",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		}, {
			name: "2 queue requests with platformName and without platformName and node with 1 slot available should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 2,
									"slotCount": 2,
									"stereotypes": "[{\"slots\": 2, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"Windows 11\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"Windows 11\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\"}",
								"{\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"Windows 11\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "Windows 11",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "1 active msedge session while asking for 2 chrome sessions should return a count of 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"msedge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"msedge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}"
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
			name: "3 queue requests browserName chrome platformName linux but 1 node has maxSessions=3 with browserName msedge should return a count of 3",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 1,
							"maxSession": 3,
							"totalSlots": 3
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 1,
									"maxSession": 3,
									"slotCount": 3,
									"stereotypes": "[{\"slots\": 3, \"stereotype\": {\"browserName\": \"msedge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}}]",
									"sessions": [
										{
											"id": "session-1",
											"capabilities": "{\"browserName\": \"msedge\", \"browserVersion\": \"91.0\", \"platformName\": \"linux\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "session request with matching browsername and no specific platformName should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"maxSession": 0,
							"nodeCount": 0,
							"totalSlots": 0
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"Windows 11\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "2 queue requests with 1 matching browsername and platformName and 1 existing slot is available should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 0,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "node-1",
									"status": "UP",
									"sessionCount": 0,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes": "[{\"slots\": 1, \"stereotype\": {\"browserName\": \"chrome\", \"browserVersion\": \"91.0\", \"platformName\": \"Windows 11\"}}]",
									"sessions": []
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"platformName\": \"Windows 11\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "Windows 11",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "2 queue requests with 1 request matching browserName and platformName but 1 existing node is busy should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "82ee33bd-390e-4dd6-aee2-06b17ecee18e",
									"status": "UP",
									"sessionCount": 2,
									"maxSession": 2,
									"slotCount": 2,
									"stereotypes": "[\n  {\n    \"slots\": 2,\n    \"stereotype\": {\n      \"browserName\": \"chrome\",\n      \"browserVersion\": \"128.0\",\n      \"goog:chromeOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n      },\n      \"se:containerName\": \"my-chrome-name-m5n8z-4br6x\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
									"sessions": [
										{
											"id": "0f9c5a941aa4d755a54b84be1f6535b1",
											"capabilities": "{\"browserName\": \"chrome\", \"platformName\": \"Windows 11\", \"browserVersion\": \"91.0\"}"
										},
										{
											"id": "0f9c5a941aa4d755a54b84be1f6535b1",
											"capabilities": "{\"browserName\": \"chrome\", \"platformName\": \"linux\", \"browserVersion\": \"91.0\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"Windows 11\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "Windows 11",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "5 queue requests with scaler parameter nodeMaxSessions is 2 should return count as 3",
			args: args{
				b: []byte(`{
				  "data": {
					"grid": {
					  "sessionCount": 0,
					  "maxSession": 0,
					  "totalSlots": 0
					},
					"nodesInfo": {
					  "nodes": []
					},
					"sessionsInfo": {
					  "sessionQueueRequests": [
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}"
					  ]
					}
				  }
				}
				`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
				nodeMaxSessions:    2,
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "5 queue requests with scaler parameter nodeMaxSessions is 3 should return count as 2",
			args: args{
				b: []byte(`{
				  "data": {
					"grid": {
					  "sessionCount": 0,
					  "maxSession": 0,
					  "totalSlots": 0
					},
					"nodesInfo": {
					  "nodes": []
					},
					"sessionsInfo": {
					  "sessionQueueRequests": [
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}",
						"{\"browserName\": \"chrome\", \"browserVersion\": \"128.0\", \"platformName\": \"Linux\"}"
					  ]
					}
				  }
				}
				`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
				nodeMaxSessions:    3,
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "5 queue requests with request matching browserName and platformName and scaler param nodeMaxSessions is 3 and existing node with 1 available slot should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 3,
							"totalSlots": 3
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "82ee33bd-390e-4dd6-aee2-06b17ecee18e",
									"status": "UP",
									"sessionCount": 2,
									"maxSession": 3,
									"slotCount": 3,
									"stereotypes": "[\n  {\n    \"slots\": 3,\n    \"stereotype\": {\n      \"browserName\": \"chrome\",\n \"platformName\": \"linux\",\n      \"browserVersion\": \"91.0\",\n      \"goog:chromeOptions\": {\n        \"binary\": \"\\u002fusr\\u002fbin\\u002fchromium\"\n      },\n      \"se:containerName\": \"my-chrome-name-m5n8z-4br6x\",\n      \"se:downloadsEnabled\": true,\n      \"se:noVncPort\": 7900,\n      \"se:vncEnabled\": true\n    }\n  }\n]",
									"sessions": [
										{
											"id": "0f9c5a941aa4d755a54b84be1f6535b1",
											"capabilities": "{\"browserName\": \"chrome\", \"platformName\": \"Linux\", \"browserVersion\": \"91.0\"}"
										},
										{
											"id": "0f9c5a941aa4d755a54b84be1f6535b1",
											"capabilities": "{\"browserName\": \"chrome\", \"platformName\": \"linux\", \"browserVersion\": \"91.0\"}"
										}
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}",
								"{\"browserName\": \"chrome\", \"platformName\": \"linux\"}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0",
				platformName:       "linux",
				nodeMaxSessions:    3,
			},
			want:    2,
			wantErr: false,
		},
		// Tests from PR: https://github.com/kedacore/keda/pull/6055
		{
			name: "sessions requests with matching browsername and platformName when setSessionsFromHub turned on and node with 1 slots matches should return count as 0",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 0,
							"maxSession": 1,
							"totalSlots": 1
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "82ee33bd-390e-4dd6-aee2-06b17ecee18e",
									"status": "UP",
									"sessionCount": 0,
									"maxSession": 1,
									"slotCount": 1,
									"stereotypes":"[{\"slots\":1,\"stereotype\":{\"browserName\":\"chrome\",\"platformName\":\"linux\"}}]",
									"sessions": []
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "4 sessions requests with matching browsername and platformName when setSessionsFromHub turned on and node with 2 slots matches should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 0,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "82ee33bd-390e-4dd6-aee2-06b17ecee18e",
									"status": "UP",
									"sessionCount": 0,
									"maxSession": 2,
									"slotCount": 2,
									"stereotypes":"[{\"slots\":2,\"stereotype\":{\"browserName\":\"chrome\",\"platformName\":\"linux\"}}]",
									"sessions": [
									]
								}
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"]
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
			name: "4 sessions requests with matching browsername and platformName when setSessionsFromHub turned on, no nodes and sessionsPerNode=2 matches should return count as 2",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 0,
							"maxSession": 0,
							"totalSlots": 0
						},
						"nodesInfo": {
							"nodes": []
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\"\n}"]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "latest",
				platformName:       "linux",
				nodeMaxSessions:    2,
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "sessions requests and active sessions with 1 matching browsername, platformName and sessionBrowserVersion should return count as 1",
			args: args{
				b: []byte(`{
					"data": {
						"grid": {
							"sessionCount": 2,
							"maxSession": 2,
							"totalSlots": 2
						},
						"nodesInfo": {
							"nodes": [
								{
									"id": "d44dcbc5-0b2c-4d5e-abf4-6f6aa5e0983c",
									"status": "UP",
									"sessionCount": 2,
									"maxSession": 2,
									"slotCount": 2,
									"stereotypes":"[{\"slots\":2,\"stereotype\":{\"browserName\":\"chrome\",\"platformName\":\"linux\"}}]",
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
							]
						},
						"sessionsInfo": {
							"sessionQueueRequests": [
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"linux\"\n}",
								"{\n  \"browserName\": \"chrome\",\n \"platformName\": \"Windows 11\",\n \"browserVersion\": \"91.0\"\n}"
							]
						}
					}
				}`),
				browserName:        "chrome",
				sessionBrowserName: "chrome",
				browserVersion:     "91.0.4472.114",
				platformName:       "linux",
			},
			want:    1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCountFromSeleniumResponse(tt.args.b, tt.args.browserName, tt.args.browserVersion, tt.args.sessionBrowserName, tt.args.platformName, tt.args.nodeMaxSessions, logr.Discard())
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
				NodeMaxSessions:    1,
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
				NodeMaxSessions:    1,
			},
		},
		{
			name: "valid url in AuthParams, browsername, and sessionbrowsername should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					AuthParams: map[string]string{
						"url":      "http://selenium-hub:4444/graphql",
						"username": "user",
						"password": "password",
					},
					TriggerMetadata: map[string]string{
						"browserName":        "MicrosoftEdge",
						"sessionBrowserName": "msedge",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                "http://selenium-hub:4444/graphql",
				Username:           "user",
				Password:           "password",
				BrowserName:        "MicrosoftEdge",
				SessionBrowserName: "msedge",
				TargetValue:        1,
				BrowserVersion:     "latest",
				PlatformName:       "linux",
				NodeMaxSessions:    1,
			},
		},
		{
			name: "valid username and password in AuthParams, url, browsername, and sessionbrowsername should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					AuthParams: map[string]string{
						"username": "username",
						"password": "password",
					},
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
				Username:           "username",
				Password:           "password",
				NodeMaxSessions:    1,
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
				NodeMaxSessions:    1,
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
				NodeMaxSessions:     1,
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
				NodeMaxSessions:     1,
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
				NodeMaxSessions:     1,
			},
		},
		{
			name: "valid url, browsername, unsafeSsl, activationThreshold, nodeMaxSessions and platformName with trigger auth params should return metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                 "http://selenium-hub:4444/graphql",
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "10",
						"platformName":        "Windows 11",
						"nodeMaxSessions":     "3",
					},
					AuthParams: map[string]string{
						"username": "user",
						"password": "password",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                 "http://selenium-hub:4444/graphql",
				Username:            "user",
				Password:            "password",
				BrowserName:         "chrome",
				SessionBrowserName:  "chrome",
				TargetValue:         1,
				ActivationThreshold: 10,
				BrowserVersion:      "91.0",
				UnsafeSsl:           true,
				PlatformName:        "Windows 11",
				NodeMaxSessions:     3,
			},
		},
		{
			name: "url in trigger auth param takes precedence over url in trigger metadata",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                 "http://invalid.dns:4444/graphql",
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "10",
						"platformName":        "Windows 11",
						"nodeMaxSessions":     "3",
					},
					AuthParams: map[string]string{
						"url":      "http://selenium-hub:4444/graphql",
						"username": "user",
						"password": "password",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                 "http://selenium-hub:4444/graphql",
				Username:            "user",
				Password:            "password",
				BrowserName:         "chrome",
				SessionBrowserName:  "chrome",
				TargetValue:         1,
				ActivationThreshold: 10,
				BrowserVersion:      "91.0",
				UnsafeSsl:           true,
				PlatformName:        "Windows 11",
				NodeMaxSessions:     3,
			},
		},
		{
			name: "auth type is not Basic and access token is provided",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"url":                 "http://selenium-hub:4444/graphql",
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "10",
						"platformName":        "Windows 11",
						"nodeMaxSessions":     "3",
					},
					AuthParams: map[string]string{
						"url":         "http://selenium-hub:4444/graphql",
						"authType":    "OAuth2",
						"accessToken": "my-access-token",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                 "http://selenium-hub:4444/graphql",
				AuthType:            "OAuth2",
				AccessToken:         "my-access-token",
				BrowserName:         "chrome",
				SessionBrowserName:  "chrome",
				TargetValue:         1,
				ActivationThreshold: 10,
				BrowserVersion:      "91.0",
				UnsafeSsl:           true,
				PlatformName:        "Windows 11",
				NodeMaxSessions:     3,
			},
		},
		{
			name: "authenticating with bearer access token",
			args: args{
				config: &scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"browserName":         "chrome",
						"browserVersion":      "91.0",
						"unsafeSsl":           "true",
						"activationThreshold": "10",
						"platformName":        "Windows 11",
						"nodeMaxSessions":     "3",
					},
					AuthParams: map[string]string{
						"url":         "http://selenium-hub:4444/graphql",
						"authType":    "Bearer",
						"accessToken": "my-access-token",
					},
				},
			},
			wantErr: false,
			want: &seleniumGridScalerMetadata{
				URL:                 "http://selenium-hub:4444/graphql",
				AuthType:            "Bearer",
				AccessToken:         "my-access-token",
				BrowserName:         "chrome",
				SessionBrowserName:  "chrome",
				TargetValue:         1,
				ActivationThreshold: 10,
				BrowserVersion:      "91.0",
				UnsafeSsl:           true,
				PlatformName:        "Windows 11",
				NodeMaxSessions:     3,
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
