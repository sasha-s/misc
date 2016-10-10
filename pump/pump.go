package pump

import "context"

type Pump struct {
	toRead    chan Interval
	toWrite   chan Interval
	blockSize int
}

// New creates a new pump.
func New(blockSize int, numBlocks int) Pump {
	toWrite := make(chan Interval, numBlocks)
	for i := 0; i < numBlocks; i++ {
		toWrite <- Interval{Start: i * blockSize, End: i*blockSize + blockSize}
	}
	return Pump{
		toRead:    make(chan Interval, numBlocks),
		toWrite:   toWrite,
		blockSize: blockSize,
	}
}

type Interval struct {
	Start int
	End   int
}

func (p Pump) StartWrite() Interval {
	return <-p.toWrite
}

func (p Pump) StartWriteCtx(ctx context.Context) (Interval, error) {
	select {
	case <-ctx.Done():
		return Interval{}, ctx.Err()
	case b := <-p.toWrite:
		return b, nil
	}
}

func (p Pump) CommitWrite(b Interval, written int) {
	if written == 0 {
		p.toWrite <- b
		return
	}
	b.End = b.Start + written
	p.toRead <- b
}

func (p Pump) StartRead() Interval {
	return <-p.toRead
}

func (p Pump) StartReadCtx(ctx context.Context) (Interval, error) {
	select {
	case <-ctx.Done():
		return Interval{}, ctx.Err()
	case b := <-p.toRead:
		return b, nil
	}
}

func (p Pump) CommitRead(b Interval) {
	b.End = b.Start + p.blockSize
	p.toWrite <- b
}

func (p Pump) CancelRead(b Interval) {
	p.toRead <- b
}
