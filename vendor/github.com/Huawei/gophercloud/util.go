package gophercloud

import (
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
	"runtime"
	"path"
)

// WaitFor polls a predicate function, once per second, up to a timeout limit.
// This is useful to wait for a resource to transition to a certain state.
// To handle situations when the predicate might hang indefinitely, the
// predicate will be prematurely cancelled after the timeout.
// Resource packages will wrap this in a more convenient function that's
// specific to a certain resource, but it can also be useful on its own.
func WaitFor(timeout int, predicate func() (bool, error)) error {
	type WaitForResult struct {
		Success bool
		Error   error
	}

	start := time.Now().Unix()

	for {
		// If a timeout is set, and that's been exceeded, shut it down.
		if timeout >= 0 && time.Now().Unix()-start >= int64(timeout) {
			return fmt.Errorf("A timeout occurred")
		}

		time.Sleep(1 * time.Second)

		var result WaitForResult
		ch := make(chan bool, 1)
		go func() {
			defer close(ch)
			satisfied, err := predicate()
			result.Success = satisfied
			result.Error = err
		}()

		select {
		case <-ch:
			if result.Error != nil {
				return result.Error
			}
			if result.Success {
				return nil
			}
			// If the predicate has not finished by the timeout, cancel it.
		case <-time.After(time.Duration(timeout) * time.Second):
			return fmt.Errorf("A timeout occurred")
		}
	}
}

// NormalizeURL is an internal function to be used by provider clients.
//
// It ensures that each endpoint URL has a closing `/`, as expected by
// ServiceClient's methods.
func NormalizeURL(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

// NormalizePathURL is used to convert rawPath to a fqdn, using basePath as
// a reference in the filesystem, if necessary. basePath is assumed to contain
// either '.' when first used, or the file:// type fqdn of the parent resource.
// e.g. myFavScript.yaml => file://opt/lib/myFavScript.yaml
func NormalizePathURL(basePath, rawPath string) (string, error) {
	u, err := url.Parse(rawPath)
	if err != nil {
		return "", err
	}
	// if a scheme is defined, it must be a fqdn already
	if u.Scheme != "" {
		return u.String(), nil
	}
	// if basePath is a url, then child resources are assumed to be relative to it
	bu, err := url.Parse(basePath)
	if err != nil {
		return "", err
	}
	var basePathSys, absPathSys string
	if bu.Scheme != "" {
		basePathSys = filepath.FromSlash(bu.Path)
		absPathSys = filepath.Join(basePathSys, rawPath)
		bu.Path = filepath.ToSlash(absPathSys)
		return bu.String(), nil
	}

	absPathSys = filepath.Join(basePath, rawPath)
	u.Path = filepath.ToSlash(absPathSys)
	if err != nil {
		return "", err
	}
	u.Scheme = "file"
	return u.String(), nil

}

// InitStructWithDefaultTag,Initialize the structure instance using the structure tag.
func InitStructWithDefaultTag(bean interface{}) {
	configType := reflect.TypeOf(bean)
	for i := 0; i < configType.Elem().NumField(); i++ {
		field := configType.Elem().Field(i)
		defaultValue := field.Tag.Get("default")
		if defaultValue == "" {
			continue
		}
		setter := reflect.ValueOf(bean).Elem().Field(i)
		switch field.Type.String() {
		case "int":
			intValue, _ := strconv.ParseInt(defaultValue, 10, 64)
			setter.SetInt(intValue)
		case "time.Duration":
			intValue, _ := strconv.ParseInt(defaultValue, 10, 64)
			setter.SetInt(intValue)
		case "string":
			setter.SetString(defaultValue)
		case "bool":
			boolValue, _ := strconv.ParseBool(defaultValue)
			setter.SetBool(boolValue)
		}
	}
}

// IsInStrSlice, Determine if the string is in the array.
func IsInStrSlice(sliceStr []string, s string) bool {
	for _, v := range sliceStr {
		if v == s {
			return true
		}
	}
	return false
}

// EnableDebug, SDK log switch defaults value is false.
var EnableDebug bool

// Logger, define the logger struct.
type Logger struct {
	DebugEnable bool `default:"false"`
}

// Debug, Format the log information and print the information to the console.
func (log *Logger) Debug(format string, v ...interface{}) {
	if log.DebugEnable {
		msg := fmt.Sprintf("[DEBUG] "+format, v...)
		writeMsg(msg)
	}
}

func writeMsg(msg string) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	_, filename := path.Split(file)
	msg = fmt.Sprintf("[%s:%s] %s", filename, strconv.FormatInt(int64(line), 10), msg)

	printMsg(msg)
}

//printMsg
func printMsg(msg string) {
	when := time.Now().Format("2006-01-02 15:04:05")
	buf := []byte(fmt.Sprintf("[%s] ", when))
	fmt.Println(string(append(append(buf, msg...))))
}

// GetLogger ,Return log initialization structure instance.
func GetLogger() (*Logger) {
	log := new(Logger)
	log.DebugEnable = EnableDebug
	return log
}
