package amqp

import "github.com/Azure/go-amqp/internal/encoding"

// DeliveryState encapsulates the various concrete delivery states.
// Use a type switch to determine the concrete delivery state.
//   - *StateAccepted
//   - *StateModified
//   - *StateReceived
//   - *StateRejected
//   - *StateReleased
type DeliveryState = encoding.DeliveryState

// StateAccepted indicates that an incoming message has been successfully processed,
// and that the receiver of the message is expecting the sender to transition the
// delivery to the accepted state at the source.
type StateAccepted = encoding.StateAccepted

// StateModifies indicates that a given transfer was not and will not be acted upon,
// and that the message SHOULD be modified in the specified ways at the node.
type StateModified = encoding.StateModified

// StateReceived indicates the furthest point in the payload of the message which the
// target will not need to have resent if the link is resumed.
type StateReceived = encoding.StateReceived

// StateRejected indicates that an incoming message is invalid and therefore unprocessable.
// The rejected outcome when applied to a message will cause the delivery-count to be
// incremented in the header of the rejected message.
type StateRejected = encoding.StateRejected

// StateReleased indicates that a given transfer was not and will not be acted upon.
type StateReleased = encoding.StateReleased
