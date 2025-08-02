package util

import (
	"reflect"
	"sync"
	"time"
)

// MergeDictoRight copies the contents of src to dest
func MergeDictoRight(src map[string]string, dest map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}

// MergeDicts creates a union of the two dicts
func MergeDicts(dic1 map[string]string, dic2 map[string]string) (resultDict map[string]string) {
	resultDict = make(map[string]string)
	MergeDictoRight(dic1, resultDict)
	MergeDictoRight(dic2, resultDict)
	return
}

// AwaitWaitGroup calls Wait on the given wait
// Returns true if the Wait() call succeeded before the timeout
// Returns false if the Wait() did not return before the timeout
func AwaitWaitGroup(wg *sync.WaitGroup, timeout time.Duration) bool {

	doneC := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneC)
	}()

	timer := time.NewTimer(timeout)
	defer func() { timer.Stop() }()

	select {
	case <-doneC:
		return true
	case <-timer.C:
		return false
	}
}

// IsInterfaceNil check if interface is nil
func IsInterfaceNil(i interface{}) bool {
	v := reflect.ValueOf(i)
	return i == nil || (v.Kind() == reflect.Ptr && v.IsNil())
}
