package proto

// Stage of query till SELECT should be executed.
type Stage byte

// Encode to buffer.
func (s Stage) Encode(b *Buffer) { b.PutUVarInt(uint64(s)) }

//go:generate go run github.com/dmarkham/enumer -type Stage -trimprefix Stage -output stage_enum.go

// StageComplete is query complete.
const (
	StageFetchColumns       Stage = 0
	StageWithMergeableState Stage = 1
	StageComplete           Stage = 2
)
