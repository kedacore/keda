package eventhub

//	MIT License
//
//	Copyright (c) Microsoft Corporation. All rights reserved.
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/mitchellh/mapstructure"

	"github.com/Azure/azure-event-hubs-go/v3/persist"
)

const (
	batchMessageFormat         uint32 = 0x80013700
	partitionKeyAnnotationName string = "x-opt-partition-key"
	sequenceNumberName         string = "x-opt-sequence-number"
	enqueueTimeName            string = "x-opt-enqueued-time"
)

type (
	// Event is an Event Hubs message to be sent or received
	Event struct {
		Data         []byte
		PartitionKey *string
		Properties   map[string]interface{}

		ID string

		message          *amqp.Message
		SystemProperties *SystemProperties

		// RawAMQPMessage is a subset of fields from the underlying AMQP message.
		// NOTE: These fields are only used when receiving events and are not sent.
		RawAMQPMessage struct {
			// Properties are standard properties for an AMQP message.
			Properties struct {
				// The identity of the user responsible for producing the message.
				// The client sets this value, and it MAY be authenticated by intermediaries.
				UserID []byte

				// This is a client-specific id that can be used to mark or identify messages
				// between clients.
				CorrelationID interface{} // uint64, UUID, []byte, or string

				// The content-encoding property is used as a modifier to the content-type.
				// When present, its value indicates what additional content encodings have been
				// applied to the application-data, and thus what decoding mechanisms need to be
				// applied in order to obtain the media-type referenced by the content-type header
				// field.
				ContentEncoding string

				// The RFC-2046 [RFC2046] MIME type for the message's application-data section
				// (body). As per RFC-2046 [RFC2046] this can contain a charset parameter defining
				// the character encoding used: e.g., 'text/plain; charset="utf-8"'.
				//
				// For clarity, as per section 7.2.1 of RFC-2616 [RFC2616], where the content type
				// is unknown the content-type SHOULD NOT be set. This allows the recipient the
				// opportunity to determine the actual type. Where the section is known to be truly
				// opaque binary data, the content-type SHOULD be set to application/octet-stream.
				ContentType string

				// A common field for summary information about the message content and purpose.
				Subject string
			}
		}
	}

	// SystemProperties are used to store properties that are set by the system.
	SystemProperties struct {
		SequenceNumber *int64     `mapstructure:"x-opt-sequence-number"` // unique sequence number of the message
		EnqueuedTime   *time.Time `mapstructure:"x-opt-enqueued-time"`   // time the message landed in the message queue
		Offset         *int64     `mapstructure:"x-opt-offset"`
		PartitionID    *int16     `mapstructure:"x-opt-partition-id"` // This value will always be nil. For information related to the event's partition refer to the PartitionKey field in this type
		PartitionKey   *string    `mapstructure:"x-opt-partition-key"`
		// Nil for messages other than from Azure IoT Hub. deviceId of the device that sent the message.
		IoTHubDeviceConnectionID *string `mapstructure:"iothub-connection-device-id"`
		// Nil for messages other than from Azure IoT Hub. Used to distinguish devices with the same deviceId, when they have been deleted and re-created.
		IoTHubAuthGenerationID *string `mapstructure:"iothub-connection-auth-generation-id"`
		// Nil for messages other than from Azure IoT Hub. Contains information about the authentication method used to authenticate the device sending the message.
		IoTHubConnectionAuthMethod *string `mapstructure:"iothub-connection-auth-method"`
		// Nil for messages other than from Azure IoT Hub. moduleId of the device that sent the message.
		IoTHubConnectionModuleID *string `mapstructure:"iothub-connection-module-id"`
		// Nil for messages other than from Azure IoT Hub. The time the Device-to-Cloud message was received by IoT Hub.
		IoTHubEnqueuedTime *time.Time `mapstructure:"iothub-enqueuedtime"`
		// Raw annotations provided on the message. Includes any additional System Properties that are not explicitly mapped.
		Annotations map[string]interface{} `mapstructure:"-"`
	}

	mapStructureTag struct {
		Name         string
		PersistEmpty bool
	}
)

// NewEventFromString builds an Event from a string message
func NewEventFromString(message string) *Event {
	return NewEvent([]byte(message))
}

// NewEvent builds an Event from a slice of data
func NewEvent(data []byte) *Event {
	return &Event{
		Data: data,
	}
}

// GetCheckpoint returns the checkpoint information on the Event
func (e *Event) GetCheckpoint() persist.Checkpoint {
	var offset string
	var enqueueTime time.Time
	var sequenceNumber int64
	if val, ok := e.message.Annotations[offsetAnnotationName]; ok {
		offset = fmt.Sprintf("%v", val)
	}

	if val, ok := e.message.Annotations[enqueueTimeName]; ok {
		enqueueTime = val.(time.Time)
	}

	if val, ok := e.message.Annotations[sequenceNumberName]; ok {
		sequenceNumber = val.(int64)
	}

	return persist.NewCheckpoint(offset, sequenceNumber, enqueueTime)
}

// GetKeyValues implements tab.Carrier
func (e *Event) GetKeyValues() map[string]interface{} {
	return e.Properties
}

// Set implements tab.Carrier
func (e *Event) Set(key string, value interface{}) {
	if e.Properties == nil {
		e.Properties = make(map[string]interface{})
	}
	e.Properties[key] = value
}

// Get will fetch a property from the event
func (e *Event) Get(key string) (interface{}, bool) {
	if e.Properties == nil {
		return nil, false
	}

	if val, ok := e.Properties[key]; ok {
		return val, true
	}
	return nil, false
}

func (e *Event) toMsg() (*amqp.Message, error) {
	msg := e.message
	if msg == nil {
		msg = amqp.NewMessage(e.Data)
	}

	msg.Properties = &amqp.MessageProperties{
		MessageID: e.ID,
	}

	if len(e.Properties) > 0 {
		msg.ApplicationProperties = make(map[string]interface{})
		for key, value := range e.Properties {
			msg.ApplicationProperties[key] = value
		}
	}

	if e.SystemProperties != nil {
		// Set the raw annotations first (they may be nil) and add the explicit
		// system properties second to ensure they're set properly.
		msg.Annotations = addMapToAnnotations(msg.Annotations, e.SystemProperties.Annotations)

		sysPropMap, err := encodeStructureToMap(e.SystemProperties)
		if err != nil {
			return nil, err
		}
		msg.Annotations = addMapToAnnotations(msg.Annotations, sysPropMap)
	}

	if e.PartitionKey != nil {
		if msg.Annotations == nil {
			msg.Annotations = make(amqp.Annotations)
		}

		msg.Annotations[partitionKeyAnnotationName] = e.PartitionKey
	}

	return msg, nil
}

func eventFromMsg(msg *amqp.Message) (*Event, error) {
	return newEvent(msg.Data[0], msg)
}

func newEvent(data []byte, msg *amqp.Message) (*Event, error) {
	event := &Event{
		Data:    data,
		message: msg,
	}

	if msg.Properties != nil {
		if id, ok := msg.Properties.MessageID.(string); ok {
			event.ID = id
		}

		event.RawAMQPMessage.Properties.UserID = msg.Properties.UserID

		if msg.Properties.Subject != nil {
			event.RawAMQPMessage.Properties.Subject = *msg.Properties.Subject
		}

		event.RawAMQPMessage.Properties.CorrelationID = msg.Properties.CorrelationID

		if msg.Properties.ContentEncoding != nil {
			event.RawAMQPMessage.Properties.ContentEncoding = *msg.Properties.ContentEncoding
		}

		if msg.Properties.ContentType != nil {
			event.RawAMQPMessage.Properties.ContentType = *msg.Properties.ContentType
		}
	}

	if msg.Annotations != nil {
		if val, ok := msg.Annotations[partitionKeyAnnotationName]; ok {
			if valStr, ok := val.(string); ok {
				event.PartitionKey = &valStr
			}
		}

		if err := mapstructure.WeakDecode(msg.Annotations, &event.SystemProperties); err != nil {
			fmt.Println("error decoding...", err)
			return event, err
		}

		// If we didn't populate any system properties, set up the struct so we
		// can put the annotations in it
		if event.SystemProperties == nil {
			event.SystemProperties = new(SystemProperties)
		}

		// Take all string-keyed annotations because the protocol reserves all
		// numeric keys for itself and there are no numeric keys defined in the
		// protocol today:
		//
		//	http://www.amqp.org/sites/amqp.org/files/amqp.pdf (section 3.2.10)
		//
		// This approach is also consistent with the behavior of .NET:
		//
		//	https://docs.microsoft.com/en-us/dotnet/api/azure.messaging.eventhubs.eventdata.systemproperties?view=azure-dotnet#Azure_Messaging_EventHubs_EventData_SystemProperties
		event.SystemProperties.Annotations = make(map[string]interface{})
		for key, val := range msg.Annotations {
			if s, ok := key.(string); ok {
				event.SystemProperties.Annotations[s] = val
			}
		}
	}

	if msg != nil {
		event.Properties = msg.ApplicationProperties
	}

	return event, nil
}

func encodeStructureToMap(structPointer interface{}) (map[string]interface{}, error) {
	valueOfStruct := reflect.ValueOf(structPointer)
	s := valueOfStruct.Elem()
	if s.Kind() != reflect.Struct {
		return nil, fmt.Errorf("must provide a struct")
	}

	encoded := make(map[string]interface{})
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.IsValid() && f.CanSet() {
			tf := s.Type().Field(i)
			tag, err := parseMapStructureTag(tf.Tag)
			if err != nil {
				return nil, err
			}

			// Skip any entries with an exclude tag
			if tag.Name == "-" {
				continue
			}

			if tag != nil {
				switch f.Kind() {
				case reflect.Ptr:
					if !f.IsNil() || tag.PersistEmpty {
						if f.IsNil() {
							encoded[tag.Name] = nil
						} else {
							encoded[tag.Name] = f.Elem().Interface()
						}
					}
				default:
					if f.Interface() != reflect.Zero(f.Type()).Interface() || tag.PersistEmpty {
						encoded[tag.Name] = f.Interface()
					}
				}
			}
		}
	}

	return encoded, nil
}

func parseMapStructureTag(tag reflect.StructTag) (*mapStructureTag, error) {
	str, ok := tag.Lookup("mapstructure")
	if !ok {
		return nil, nil
	}

	mapTag := new(mapStructureTag)
	split := strings.Split(str, ",")
	mapTag.Name = strings.TrimSpace(split[0])

	if len(split) > 1 {
		for _, tagKey := range split[1:] {
			switch tagKey {
			case "persistempty":
				mapTag.PersistEmpty = true
			default:
				return nil, fmt.Errorf("key %q is not understood", tagKey)
			}
		}
	}
	return mapTag, nil
}

func addMapToAnnotations(a amqp.Annotations, m map[string]interface{}) amqp.Annotations {
	if a == nil && len(m) > 0 {
		a = make(amqp.Annotations)
	}
	for key, val := range m {
		a[key] = val
	}
	return a
}
