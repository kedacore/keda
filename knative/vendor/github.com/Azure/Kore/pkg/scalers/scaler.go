package scalers

type Scaler interface {
	GetScaleDecision() (int32, error)
}
