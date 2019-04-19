package servicebus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-amqp-common-go/log"
	"github.com/Azure/azure-amqp-common-go/rpc"
	"go.opencensus.io/trace"
	"pack.ag/amqp"
)

type (
	lockRenewer interface {
		entityConnector
		lockMutex() *sync.Mutex
	}
)

func renewLocks(ctx context.Context, lr lockRenewer, messages ...*Message) error {
	span, ctx := startConsumerSpanFromContext(ctx, "sb.RenewLocks")
	defer span.Finish()

	lockTokens := make([]amqp.UUID, 0, len(messages))
	for _, m := range messages {
		if m.LockToken == nil {
			log.For(ctx).Error(fmt.Errorf("failed: message has nil lock token, cannot renew lock"), trace.StringAttribute("messageId", m.ID))
			continue
		}

		amqpLockToken := amqp.UUID(*m.LockToken)
		lockTokens = append(lockTokens, amqpLockToken)
	}

	if len(lockTokens) < 1 {
		log.For(ctx).Info("no lock tokens present to renew")
		return nil
	}

	lr.lockMutex().Lock()
	defer lr.lockMutex().Unlock()

	renewRequestMsg := &amqp.Message{
		ApplicationProperties: map[string]interface{}{
			operationFieldName: lockRenewalOperationName,
		},
		Value: map[string]interface{}{
			lockTokensFieldName: lockTokens,
		},
	}

	entityManagementAddress := lr.ManagementPath()
	conn, err := lr.connection(ctx)
	if err != nil {
		return err
	}

	rpcLink, err := rpc.NewLink(conn, entityManagementAddress)
	if err != nil {
		return err
	}

	response, err := rpcLink.RetryableRPC(ctx, 3, 1*time.Second, renewRequestMsg)
	if err != nil {
		return err
	}

	if response.Code != 200 {
		return fmt.Errorf("error renewing locks: %v", response.Description)
	}

	return nil
}
