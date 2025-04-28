package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type CellIndex int

type BoardSpec struct {
	size        int
	blockWidth  int
	blockHeight int
	blocks      []*Group
	cols        []*Group
	rows        []*Group
}

type Board struct {
	cells []Cell
	spec  *BoardSpec
	quiet bool
}

// Creates an empty sudoku board with blocks of the specified
// dimension. Note that these dimensions determine the overall
// layout of the board.
func NewBoard(blockWidth, blockHeight int) *Board {
	// first, create all our cells
	board := &Board{}
	spec := &BoardSpec{}
	board.spec = spec

	spec.size = blockWidth * blockHeight
	spec.blockWidth = blockWidth
	spec.blockHeight = blockHeight
	board.cells = make([]Cell, spec.size*spec.size)

	var blocks, cols, rows []*Group

	for y := 0; y < blockWidth; y++ {
		for x := 0; x < blockHeight; x++ {
			blocks = append(blocks, NewBlockGroup(board, x, y, blockWidth, blockHeight))
		}
	}

	for x := 0; x < spec.size; x++ {
		cols = append(cols, NewColumnGroup(board, x))
	}

	for y := 0; y < spec.size; y++ {
		rows = append(rows, NewRowGroup(board, y))
	}

	spec.blocks = blocks
	spec.cols = cols
	spec.rows = rows

	return board
}

func (b *Board) Duplicate() *Board {
	var nb Board

	nb.cells = make([]Cell, len(b.cells))
	copy(nb.cells, b.cells)
	nb.spec = b.spec
	nb.quiet = b.quiet
	return &nb
}

func (b *Board) Size() int {
	return b.spec.size
}

func (b *Board) Cell(ci CellIndex) *Cell {
	return &b.cells[ci]
}

func (b *Board) CellAt(x, y int) *Cell {
	return &b.cells[b.CellIndex(x, y)]
}

func (b *Board) CellIndex(x, y int) CellIndex {
	return CellIndex(x + y*b.Size())
}

func (b *Board) Coords(c *Cell) (x, y, z int) {
	for i := range b.cells {
		if c == &b.cells[i] {
			return b.IndexToCoords(CellIndex(i))
		}
	}
	panic(fmt.Errorf("couldn't find cell %+v", c))
}

// IndexToCoords returns the column index, row index, and
// block index of the cell with the given CellIndex
func (b *Board) IndexToCoords(ci CellIndex) (x, y, z int) {
	x, y = int(ci)%b.Size(), int(ci)/b.Size()
	z = x/b.spec.blockWidth + y/b.spec.blockHeight*b.spec.blockHeight
	return x, y, z
}

func (b Board) Groups() []*Group {
	return append(b.spec.blocks, append(b.spec.cols, b.spec.rows...)...)
}

func (b *Board) GroupsContaining(ci CellIndex) []*Group {
	result := make([]*Group, 0, 3)

	x, y, z := b.IndexToCoords(ci)

	result = append(result, b.spec.cols[x])
	result = append(result, b.spec.rows[y])
	result = append(result, b.spec.blocks[z])
	return result
}

func (b *Board) SetValue(depth, reason string, x, y, val int) error {
	ci := b.CellIndex(x, y)

	// already set to the given value? just exit
	if v, set := b.Cell(ci).GetValue(); set && val == v {
		return nil
	}

	if !b.Cell(ci).CanTake(val) {
		x, y, _ := b.IndexToCoords(ci)
		return fmt.Errorf("tried to set value on %d not legal at (%d,%d)", val, x, y)
	}

	if !b.quiet {
		fmt.Printf("%s%s: (%d,%d) -> %d\n", depth, reason, x, y, val)
	}
	if err := b.Cell(ci).SetValue(val); err != nil {
		return err
	}

	// In order to avoid corruption of the board, we need to handle
	// prohibition in two steps: first, we mark all the effected cells,
	// then, we deal with the fallout from any of those markings
	var marked []CellIndex
	for _, g := range b.GroupsContaining(ci) {
		if g.Contains(ci) {
			for _, i := range g.Indices() {
				if i != ci && b.Cell(i).CanTake(val) {
					x, y, _ := b.IndexToCoords(i)
					if err := b.ProhibitValue(depth+" ", "excluding because of set value", x, y, val); err != nil {
						return err
					}
					marked = append(marked, ci)
				} else {
					// this won't impact anything about how the puzzle gets solved,
					// but it does ensure our candidate lists remain in sync
					b.Cell(i).Prohibit(val)
				}
			}
		}
	}

	for _, ci := range marked {
		candidates := b.Cell(ci).Candidates(b.Size())
		if len(candidates) == 1 {
			x, y, _ := b.IndexToCoords(ci)
			if err := b.SetValue(depth+" ", "only one left after prohibition", x, y, candidates[0]); err != nil {
				return err
			}
		}
	}
	return b.IsValid()
}

// UnsetValue clears the value in the specified cell then
// resets and recalculates prohibition/candidate lists each
// impacted cell.
func (b *Board) UnsetValue(ci CellIndex) {
	cell := b.Cell(ci)
	if _, isSet := cell.GetValue(); isSet {
		cell.value = 0
		for _, g := range b.GroupsContaining(ci) {
			for _, oci := range g.Indices() {
				b.Cell(oci).ClearProhibitions()
				for _, og := range b.GroupsContaining(oci) {
					for _, val := range og.SetValues(b) {
						b.Cell(oci).Prohibit(val)
					}
				}
			}
		}
	}
}

func (b *Board) ProhibitValue(depth, reason string, x, y, val int) error {
	cell := b.CellAt(x, y)
	if cell.CanTake(val) {
		if !b.quiet {
			fmt.Printf("%s%s: (%d,%d) cannot be %d\n", depth, reason, x, y, val)
		}
	}
	if err := cell.Prohibit(val); err != nil {
		return err
	}

	return nil
}

func (b *Board) String() string {
	var buf bytes.Buffer

	for y := 0; y < b.Size(); y++ {
		for x := 0; x < b.Size(); x++ {
			cell := b.CellAt(x, y)
			if val, ok := cell.GetValue(); ok {
				fmt.Fprintf(&buf, "%d ", val)
			} else {
				fmt.Fprintf(&buf, "- ")
			}
			if (x+1)%b.spec.blockWidth == 0 {
				fmt.Fprintf(&buf, "\t")
			}
		}
		if (y+1)%b.spec.blockHeight == 0 {
			fmt.Fprintf(&buf, "\n")
		}
		fmt.Fprintf(&buf, "\n")
	}
	return buf.String()
}

func NewBoardFromBuffer(blockWidth, blockHeight int, input io.Reader) (*Board, error) {
	scanner := bufio.NewScanner(input)
	board := NewBoard(blockWidth, blockHeight)
	y := 0

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		for x, c := range line {
			if c == '-' || c == ' ' {
				continue
			}
			val := c - '0'
			if err := board.SetValue("", "initial value", x, y, int(val)); err != nil {
				return nil, err
			}
		}
		y++
	}
	return board, nil
}

func (b *Board) IsSolved() bool {
	for _, c := range b.cells {
		if _, set := c.GetValue(); !set {
			return false
		}
	}
	return true
}

func (b *Board) Unfilled() *Group {
	var cells []CellIndex
	for ci, c := range b.cells {
		if _, set := c.GetValue(); !set {
			cells = append(cells, CellIndex(ci))
		}
	}
	return NewGroup(cells)
}

func (b *Board) Filled() *Group {
	var cells []CellIndex
	for ci, c := range b.cells {
		if _, set := c.GetValue(); set {
			cells = append(cells, CellIndex(ci))
		}
	}
	return NewGroup(cells)
}

func within(ci int, list []int) bool {
	for _, member := range list {
		if ci == member {
			return true
		}
	}
	return false
}

func (b *Board) Unsolved() string {
	var buf bytes.Buffer

	for y := 0; y < b.Size(); y++ {
		for x := 0; x < b.Size(); x++ {
			c := b.CellAt(x, y)
			if c.Filled() {
				continue
			}
			fmt.Fprintf(&buf, "(%d,%d): %+v\n", x, y, c.Candidates(b.Size()))
		}
	}
	return buf.String()
}

func (b *Board) IsValid() error {
	for _, g := range append(b.spec.blocks, append(b.spec.rows, b.spec.cols...)...) {
		seen := make(map[int]CellIndex)
		for _, ci := range g.Indices() {
			c := b.Cell(ci)
			if v, set := c.GetValue(); set {
				if _, found := seen[v]; found {
					x1, y1, _ := b.IndexToCoords(ci)
					x2, y2, _ := b.IndexToCoords(seen[v])
					if !b.quiet {
						fmt.Println("invalid board detected")
					}
					panic(fmt.Errorf("(%d,%d) and (%d,%d) are both %d", x1, y1, x2, y2, v))
				} else {
					seen[v] = ci
				}
			}
		}
	}

	for ci, c := range b.cells {
		if !c.IsSet() {
			if len(c.Candidates(b.Size())) == 0 {
				if !b.quiet {
					fmt.Println("invalid board detected")
				}
				x, y, _ := b.IndexToCoords(CellIndex(ci))
				return fmt.Errorf("(%d,%d) is unfilled and has no candidates", x, y)
			}
		}
	}

	for ci, c := range b.cells {
		if c.IsSet() {
			continue
		}
		for val := 1; val <= b.Size(); val++ {
			found := false
			var index CellIndex
			for _, g := range b.GroupsContaining(CellIndex(ci)) {
				if index, found = g.FindSetValue(val, b); found {
					break
				}
			}
			if found && c.CanTake(val) {
				x, y, _ := b.IndexToCoords(CellIndex(ci))
				x2, y2, _ := b.IndexToCoords(index)
				panic(fmt.Errorf("(%d,%d) says it can take %d but (%d,%d)=%d", x, y, val, x2, y2, val))
			}

			/* Attenpting to validate the consistency of the prohibition list but
			this isn't correct: things can be prohibited for more complex reasons.

			If something isn't on the prohibition list, but should be, *that* is an error.


			if !found && !c.CanTake(val) {
				x, y := b.IndexToCoords(CellIndex(ci))
				panic(fmt.Errorf("(%d,%d) says it cannot take %d but it can", x, y, val))
			}
			*/
		}
	}

	return nil
}

func (b *Board) Validate() {
	if err := b.IsValid(); err != nil {
		panic(fmt.Errorf("invalid board: %e", err))
	}
}
