/*
 The MIT License

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package influxdb3

import (
	"fmt"
	"net/url"
)

// ReplaceURLProtocolWithPort removes the "http://" or "https://" protocol from the given URL and replaces it with the port number.
// Currently, Apache Arrow does not support the "http://" or "https://" protocol in the URL, so this function is used to remove it.
// If a port number is already present in the URL, only the protocol is removed.
// The function also returns a boolean value indicating whether the communication is safe or unsafe.
// - If the URL starts with "https://", the communication is considered safe, and the returned boolean value will be true.
// - If the URL starts with "http://", the communication is considered unsafe, and the returned boolean value will be false.
// - If the URL does not start with either "http://" or "https://", the returned boolean value will be nil.
//
// Parameters:
//   - url: The URL to process.
//
// Returns:
//   - The modified URL with the protocol replaced by the port.
//   - A boolean value indicating the safety of communication (true for safe, false for unsafe) or nil if not detected.
func ReplaceURLProtocolWithPort(serverURL string) (string, bool) {
	u, _ := url.Parse(serverURL)
	var safe bool
	if u.Scheme == schemeHTTPS {
		safe = true
	}
	serverURL = fmt.Sprintf("%s:%s%s", u.Hostname(), port(u), u.Path)
	return serverURL, safe
}

func port(u *url.URL) string {
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == schemeHTTPS {
			port = "443"
		}
	}
	return port
}
