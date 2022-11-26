package kusto

// statemachine.go provides statemachines for interpreting frame streams for varying Kusto options.
// Based on the standard Go statemachine design by Rob Pike.

import (
	"context"
	"sync"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
	v1 "github.com/Azure/azure-kusto-go/kusto/internal/frames/v1"
	v2 "github.com/Azure/azure-kusto-go/kusto/internal/frames/v2"
)

// stateFn represents a function that executes at a given state.
type stateFn func() (stateFn, error)

// stateMachine provides a state machine for executing a of of well defined states.
type stateMachine interface {
	// start starts the stateMachine and returns either the next state to run, an error, or nil, nil.
	start() (stateFn, error)
	rowIter() *RowIterator
}

// runSM runs a stateMachine to its conclusion.
func runSM(sm stateMachine) {
	defer close(sm.rowIter().inRows)

	var fn = sm.start
	var err error
	for {
		fn, err = fn()
		switch {
		case err != nil:
			sm.rowIter().inErr <- send{inErr: err} // Unique case, don't send a WaitGroup (also means, design needs to be fixed)
			return
		case fn == nil && err == nil:
			return
		}
	}
}

// nonProgressiveSM implements a stateMachine that processes Kusto data that is not non-streaming.
type nonProgressiveSM struct {
	op            errors.Op
	iter          *RowIterator
	in            chan frames.Frame
	columnSetOnce sync.Once
	ctx           context.Context
	hasCompletion bool

	wg *sync.WaitGroup // Used to know when everything has finished
}

func (d *nonProgressiveSM) start() (stateFn, error) {
	return d.process, nil
}

func (d *nonProgressiveSM) rowIter() *RowIterator {
	return d.iter
}

func (d *nonProgressiveSM) process() (sf stateFn, err error) {
	select {
	case <-d.ctx.Done():
		return nil, d.ctx.Err()
	case fr, ok := <-d.in:
		if !ok {
			d.wg.Wait()

			if !d.hasCompletion {
				return nil, errors.ES(d.op, errors.KInternal, "non-progressive stream did not have DataSetCompletion frame")
			}
			return nil, nil
		}

		if d.hasCompletion {
			return nil, errors.ES(d.op, errors.KInternal, "saw a DataSetCompletion frame, then received a %T frame", fr)
		}

		switch table := fr.(type) {
		case v2.DataTable:
			d.wg.Add(1)
			switch table.TableKind {
			case frames.PrimaryResult:
				d.columnSetOnce.Do(func() {
					d.wg.Add(1) // We add here as well because two things are sent in this case statement.
					d.iter.inColumns <- send{inColumns: table.Columns, wg: d.wg}
				})

				select {
				case <-d.ctx.Done():
					return nil, d.ctx.Err()
				case d.iter.inRows <- send{inRows: table.KustoRows, inRowErrors: table.RowErrors, wg: d.wg}:
				}
			default:
				select {
				case <-d.ctx.Done():
					return nil, d.ctx.Err()
				case d.iter.inNonPrimary <- send{inNonPrimary: table, wg: d.wg}:
				}
			}
		case frames.Error:
			return nil, table
		case v2.DataSetCompletion:
			d.wg.Add(1)

			select {
			case <-d.ctx.Done():
				return nil, d.ctx.Err()
			case d.iter.inCompletion <- send{inCompletion: table, wg: d.wg}:
			}
			d.hasCompletion = true
		}
	}
	return d.process, nil
}

/*
progressiveSM implements a stateMachine that handles progressive streaming Kusto data. Progressive streams really add
support for giving progress to APIs that want to know how far they are into a set of results. A progressive stream
should have the following structure and anything outside this should cause an error:

Either:

	TableHeader
	Progress
	DataTable
	DataTableCompletion

	If TableHeader:
		Followed by Fragment
		Followed by a Fragment or Completion
			If Fragment, conintue until Completion
			If Completion, loop()

	If Progress:
		Followed by anything
		loop()

	If DataTable:
		Must be Non-Primary
		loop()

	If DataTableCompletion:
		End, but we had to have had a TableHeader
*/
type progressiveSM struct {
	op            errors.Op
	iter          *RowIterator
	in            chan frames.Frame
	columnSetOnce sync.Once
	ctx           context.Context

	currentHeader *v2.TableHeader
	currentFrame  frames.Frame
	nonPrimary    *v2.DataTable

	wg *sync.WaitGroup
}

func (p *progressiveSM) start() (stateFn, error) {
	return p.nextFrame, nil
}

func (p *progressiveSM) rowIter() *RowIterator {
	return p.iter
}

func (p *progressiveSM) nextFrame() (stateFn, error) {
	// These are two separate select cases since we always want to check for context cancellation first, otherwise order is not guaranteed.

	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	default:
	}

	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case fr, ok := <-p.in:
		if !ok {
			return nil, errors.ES(p.op, errors.KInternal, "received a table stream that did not finish before our input channel, this is usually a return size or time limit")
		}

		p.currentFrame = fr
		switch table := fr.(type) {
		case v2.DataTable:
			return p.dataTable, nil
		case v2.DataSetCompletion:
			return p.dataSetCompletion, nil
		case v2.TableHeader:
			return p.tableHeader, nil
		case v2.TableFragment:
			return p.fragment, nil
		case v2.TableProgress:
			return p.progress, nil
		case v2.TableCompletion:
			return p.completion, nil
		case frames.Error:
			return nil, table
		default:
			return nil, errors.ES(p.op, errors.KInternal, "received an unknown frame in a progressive table stream we didn't understand: %T", table)
		}
	}
}

func (p *progressiveSM) dataTable() (stateFn, error) {
	if p.currentHeader != nil {
		return nil, errors.ES(p.op, errors.KInternal, "received a DatTable between a tableHeader and TableCompletion")
	}
	table := p.currentFrame.(v2.DataTable)
	if table.TableKind == frames.PrimaryResult {
		return nil, errors.ES(p.op, errors.KInternal, "progressive stream had dataTable with Kind == PrimaryResult")
	}

	p.wg.Add(1)

	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case p.iter.inNonPrimary <- send{inNonPrimary: table, wg: p.wg}:
	}

	return p.nextFrame, nil
}

func (p *progressiveSM) dataSetCompletion() (stateFn, error) {
	p.wg.Add(1)

	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case p.iter.inCompletion <- send{inCompletion: p.currentFrame.(v2.DataSetCompletion), wg: p.wg}:
	}

	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case frame, ok := <-p.in:
		if !ok {
			p.wg.Wait()
			return nil, nil
		}
		return nil, errors.ES(p.op, errors.KInternal, "received a dataSetCompletion frame and then a %T frame", frame)
	}
}

func (p *progressiveSM) tableHeader() (stateFn, error) {
	table := p.currentFrame.(v2.TableHeader)
	p.currentHeader = &table
	if p.currentHeader.TableKind == frames.PrimaryResult {
		p.columnSetOnce.Do(func() {
			p.wg.Add(1)
			p.iter.inColumns <- send{inColumns: table.Columns, wg: p.wg}
		})
	} else {
		p.nonPrimary = &v2.DataTable{
			Base:      v2.Base{FrameType: frames.TypeDataTable},
			TableID:   p.currentHeader.TableID,
			TableKind: p.currentHeader.TableKind,
			TableName: p.currentHeader.TableName,
			Columns:   p.currentHeader.Columns,
		}
	}

	return p.nextFrame, nil
}

func (p *progressiveSM) fragment() (stateFn, error) {
	if p.currentHeader == nil {
		return nil, errors.ES(p.op, errors.KInternal, "received a TableFragment without a tableHeader")
	}

	if p.currentHeader.TableKind == frames.PrimaryResult {
		table := p.currentFrame.(v2.TableFragment)

		p.wg.Add(1)
		select {
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		case p.iter.inRows <- send{inRows: table.KustoRows, inRowErrors: table.RowErrors, inTableFragmentType: table.TableFragmentType, wg: p.wg}:
		}
	} else {
		p.nonPrimary.Rows = append(p.nonPrimary.Rows, p.currentFrame.(v2.TableFragment).Rows...)
	}
	return p.nextFrame, nil
}

func (p *progressiveSM) progress() (stateFn, error) {
	if p.currentHeader == nil {
		return nil, errors.ES(p.op, errors.KInternal, "received a TableProgress without a tableHeader")
	}
	p.wg.Add(1)
	p.iter.inProgress <- send{inProgress: p.currentFrame.(v2.TableProgress), wg: p.wg}
	return p.nextFrame, nil
}

func (p *progressiveSM) completion() (stateFn, error) {
	if p.currentHeader == nil {
		return nil, errors.ES(p.op, errors.KInternal, "received a TableCompletion without a tableHeader")
	}
	if p.currentHeader.TableKind == frames.PrimaryResult {
		// Do nothing here.
	} else {
		p.wg.Add(1)
		p.iter.inNonPrimary <- send{inNonPrimary: *p.nonPrimary, wg: p.wg}
	}
	p.nonPrimary = nil
	p.currentHeader = nil
	p.currentFrame = nil

	return p.nextFrame, nil
}

// v1SM implements a stateMachine that handles v1 MGMT streaming Kusto data.
type v1SM struct {
	op            errors.Op
	iter          *RowIterator
	in            chan frames.Frame
	columnSetOnce sync.Once
	ctx           context.Context

	currentTable v1.DataTable
	tables       []v1.DataTable

	receivedDT bool

	wg *sync.WaitGroup
}

type TableOfContents struct {
	Ordinal    int64
	Kind       string
	Name       string
	Id         string
	PrettyName string
}

func (p *v1SM) start() (stateFn, error) {
	return p.nextFrame, nil
}

func (p *v1SM) rowIter() *RowIterator {
	return p.iter
}

func (p *v1SM) nextFrame() (stateFn, error) {
	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case fr, ok := <-p.in:
		if !ok {
			if len(p.tables) == 0 {
				return p.done, nil
			}
			if len(p.tables) <= 2 {
				p.currentTable = p.tables[0]
			} else {
				p.currentTable = p.tables[len(p.tables)-1]
				return p.tableOfContents, nil
			}
			return p.dataTable, nil
		}
		switch tbl := fr.(type) {
		case v1.DataTable:
			p.tables = append(p.tables, tbl)
			return p.nextFrame, nil
		case frames.Error:
			return nil, tbl
		default:
			return nil, errors.ES(p.op, errors.KInternal, "received an unknown frame in a v1 table stream we didn't understand: %T", tbl)
		}
	}
}
func (p *v1SM) done() (stateFn, error) {
	if !p.receivedDT {
		return nil, errors.ES(p.op, errors.KInternal, "received a table stream that did not finish before our input channel, this is usually a return size or time limit")
	}
	p.wg.Wait()

	return nil, nil
}

func (p *v1SM) tableOfContents() (stateFn, error) {
	tableOfContents := p.currentTable
	columns, err := tableOfContents.DataTypes.ToColumns()
	if err != nil {
		return nil, err
	}

	current := TableOfContents{}
	for _, kustoRow := range tableOfContents.KustoRows {
		row := table.Row{ColumnTypes: columns, Values: kustoRow, Op: p.op}
		err := row.ToStruct(&current)
		if err != nil {
			return nil, err
		}

		kind := frames.TableKind(current.Kind)
		if kind == frames.QueryResult {
			p.currentTable = p.tables[current.Ordinal]
			if _, err := p.dataTable(); err != nil {
				return nil, err
			}
		}
	}
	return p.done, nil
}

func (p *v1SM) dataTable() (stateFn, error) {
	var err error
	currentTable := p.currentTable

	p.columnSetOnce.Do(func() {
		var cols table.Columns
		cols, err = currentTable.DataTypes.ToColumns()
		if err != nil {
			return
		}
		p.wg.Add(1)
		p.iter.inColumns <- send{inColumns: cols, wg: p.wg}
	})

	p.wg.Add(1)
	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	case p.iter.inRows <- send{inRows: currentTable.KustoRows, inRowErrors: currentTable.RowErrors, wg: p.wg}:
		p.receivedDT = true
	}

	return p.done, nil
}
