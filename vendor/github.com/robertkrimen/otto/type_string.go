package otto

import (
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

type stringObjecter interface {
	Length() int
	At(at int) rune
	String() string
}

type stringASCII string

func (str stringASCII) Length() int {
	return len(str)
}

func (str stringASCII) At(at int) rune {
	return rune(str[at])
}

func (str stringASCII) String() string {
	return string(str)
}

type stringWide struct {
	string  string
	value16 []uint16
}

func (str stringWide) Length() int {
	if str.value16 == nil {
		str.value16 = utf16.Encode([]rune(str.string))
	}
	return len(str.value16)
}

func (str stringWide) At(at int) rune {
	if str.value16 == nil {
		str.value16 = utf16.Encode([]rune(str.string))
	}
	return rune(str.value16[at])
}

func (str stringWide) String() string {
	return str.string
}

func newStringObject(str string) stringObjecter {
	for i := range len(str) {
		if str[i] >= utf8.RuneSelf {
			goto wide
		}
	}

	return stringASCII(str)

wide:
	return &stringWide{
		string: str,
	}
}

func stringAt(str stringObjecter, index int) rune {
	if 0 <= index && index < str.Length() {
		return str.At(index)
	}
	return utf8.RuneError
}

func (rt *runtime) newStringObject(value Value) *object {
	str := newStringObject(value.string())

	obj := rt.newClassObject(classStringName)
	obj.defineProperty(propertyLength, intValue(str.Length()), 0, false)
	obj.objectClass = classString
	obj.value = str
	return obj
}

func (o *object) stringValue() stringObjecter {
	if str, ok := o.value.(stringObjecter); ok {
		return str
	}
	return nil
}

func stringEnumerate(obj *object, all bool, each func(string) bool) {
	if str := obj.stringValue(); str != nil {
		length := str.Length()
		for index := range length {
			if !each(strconv.FormatInt(int64(index), 10)) {
				return
			}
		}
	}
	objectEnumerate(obj, all, each)
}

func stringGetOwnProperty(obj *object, name string) *property {
	if prop := objectGetOwnProperty(obj, name); prop != nil {
		return prop
	}
	// TODO Test a string of length >= +int32 + 1?
	if index := stringToArrayIndex(name); index >= 0 {
		if chr := stringAt(obj.stringValue(), int(index)); chr != utf8.RuneError {
			return &property{stringValue(string(chr)), 0}
		}
	}
	return nil
}
