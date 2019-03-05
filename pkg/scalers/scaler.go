package scalers

import (
	"context"
)

type Scaler interface {
	GetScaleDecision(ctx context.Context) (int32, error)
}
