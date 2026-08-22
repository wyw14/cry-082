package telemetry

type BatchItemState string

const (
	BatchItemPending  BatchItemState = "pending"
	BatchItemAccepted BatchItemState = "accepted"
	BatchItemRejected BatchItemState = "rejected"
	BatchItemSkipped  BatchItemState = "skipped"
)

type BatchDisposition struct {
	items      []BatchItemState
	accepted   int
	rejected   int
	haltAt     int
	terminated bool
}

func NewBatchDisposition(size int) *BatchDisposition {
	if size < 0 {
		size = 0
	}
	items := make([]BatchItemState, size)
	for index := range items {
		items[index] = BatchItemPending
	}
	return &BatchDisposition{items: items, haltAt: -1}
}

func (b *BatchDisposition) Accept(index int) {
	if b == nil || index < 0 || index >= len(b.items) || b.items[index] != BatchItemPending {
		return
	}
	b.items[index] = BatchItemAccepted
	b.accepted++
}

func (b *BatchDisposition) Reject(index int) {
	if b == nil || index < 0 || index >= len(b.items) || b.items[index] != BatchItemPending {
		return
	}
	b.items[index] = BatchItemRejected
	b.rejected++
	b.haltAt = index
	b.terminated = true
	for next := index + 1; next < len(b.items); next++ {
		if b.items[next] == BatchItemPending {
			b.items[next] = BatchItemSkipped
		}
	}
}

func (b *BatchDisposition) Halted() bool {
	return b != nil && b.terminated
}

func (b *BatchDisposition) State(index int) BatchItemState {
	if b == nil || index < 0 || index >= len(b.items) {
		return BatchItemSkipped
	}
	return b.items[index]
}

func (b *BatchDisposition) Totals() (accepted, rejected int) {
	if b == nil {
		return 0, 0
	}
	return b.accepted, b.rejected
}
