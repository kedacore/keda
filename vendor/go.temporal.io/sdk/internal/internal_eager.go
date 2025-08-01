package internal

// eagerWorker is the minimal worker interface needed for eager activities and workflows
type eagerWorker interface {
	// tryReserveSlot tries to reserver a task slot on the worker without blocking
	// caller is expected to release the slot with releaseSlot
	tryReserveSlot() *SlotPermit
	// releaseSlot release a task slot acquired by tryReserveSlot
	releaseSlot(permit *SlotPermit, reason SlotReleaseReason)
	// pushEagerTask pushes a new eager workflow task to the workers task queue.
	// should only be called with a reserved slot.
	pushEagerTask(task eagerTask)
}
