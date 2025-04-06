package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strings"
)

type CellIndex int

type Board struct {
	cells       []Cell
	width       int
	height      int
	blockWidth  int
	blockHeight int
	blocks      []Group
	cols        []Group
	rows        []Group
}

func NewBoard(blockWidth, blockHeight, blockCountHoriz, blockCountVert int) *Board {
	// first, create all our cells
	board := &Board{}
	board.width = blockWidth * blockCountHoriz
	board.height = blockHeight * blockCountVert
	board.blockWidth = blockWidth
	board.blockHeight = blockHeight
	board.cells = make([]Cell, board.width*board.height)

	var blocks, cols, rows []Group

	for x := 0; x < blockCountHoriz; x++ {
		for y := 0; y < blockCountVert; y++ {
			blocks = append(blocks, NewBlockGroup(board, x, y, blockWidth, blockHeight))
		}
	}

	for x := 0; x < board.width; x++ {
		cols = append(cols, NewColumnGroup(board, x))
	}

	for y := 0; y < board.height; y++ {
		rows = append(rows, NewRowGroup(board, y))
	}

	board.blocks = blocks
	board.cols = cols
	board.rows = rows

	return board
}

// Given a board in any state, returns a new board with all of its cells filled (or error)
func Fill(board *Board) (*Board, error) {
	// The basic idea here is a hyrbid of brute-force and heuristic.
	// At each step, we randomly pick an open cell, and iterate over each
	// of its candidates, trying each one in turn. After filling a cell
	// with a candidate, we attempt to solve the puzzle heuristically. This
	// leads to three possible outcomes:
	// 1. An error -- there's an inherent conflict; the candidate we picked leads
	// to a puzzle that cannot be solved.
	// 2. An unsolved puzzle: the candidate itself is okay, but the solver cannot
	// generate a unique solution.
	// 3. A solved puzzle: the solver was able to complete the puzzle with a unique
	// solution.
	// In the first case, we simply backtrack and try the next candidate.
	// In the second case, we pick another random cell and continue.
	// In the third case, we've succeeded, so we return the filled board.

	var fillNextCell func(*Board) (*Board, error)
	var tryCandidate func(*Board, CellIndex, int) (*Board, error)

	fillNextCell = func(board *Board) (*Board, error) {
		// Pick an unfilled cell randomly
		ci := board.Unfilled().PickOne()
		candidates := board.Cell(ci).Possibilities(board.height)
		rand.Shuffle(len(candidates), func(x, y int) { candidates[x], candidates[y] = candidates[y], candidates[x] })
		for _, candidate := range candidates {
			if nb, err := tryCandidate(board.Duplicate(), ci, candidate); err == nil {
				return nb, nil
			}
		}
		// we tried all candidates for cell but none succeeded
		return nil, fmt.Errorf("no candidate worked for cell %d", int(ci))
	}

	tryCandidate = func(board *Board, ci CellIndex, value int) (*Board, error) {
		x, y := board.IndexToCoords(ci)
		if err := board.SetValue("", "random pick", x, y, value); err != nil {
			return nil, err
		}
		// we were able to set a value; now try to use solver to fill in some more
		if err := board.Solve(); err != nil {
			return nil, err
		}
		if board.IsSolved() {
			return board, nil
		}
		// still not solved; try filling more values randomly
		return fillNextCell(board)
	}

	return fillNextCell(board.Duplicate())

}

func (b *Board) Duplicate() *Board {
	var nb Board

	nb.cells = make([]Cell, len(b.cells))
	copy(nb.cells, b.cells)
	nb.width = b.width
	nb.height = b.height
	nb.blockWidth = b.blockWidth
	nb.blockHeight = b.blockHeight
	nb.blocks = make([]Group, len(b.blocks))
	nb.cols = make([]Group, len(b.cols))
	nb.rows = make([]Group, len(b.rows))
	copy(nb.blocks, b.blocks)
	copy(nb.cols, b.cols)
	copy(nb.rows, b.rows)
	return &nb
}

func (b *Board) MaxVal() int {
	return b.width
}

func (b *Board) Cell(ci CellIndex) *Cell {
	return &b.cells[ci]
}

func (b *Board) CellAt(x, y int) *Cell {
	return &b.cells[b.CellIndex(x, y)]
}

func (b *Board) CellIndex(x, y int) CellIndex {
	return CellIndex(x + y*b.width)
}

func (b *Board) Coords(c *Cell) (x int, y int) {
	for i := range b.cells {
		if c == &b.cells[i] {
			return b.IndexToCoords(CellIndex(i))
		}
	}
	panic(fmt.Errorf("couldn't find cell %+v", c))
}

func (b *Board) IndexToCoords(ci CellIndex) (x, y int) {
	return int(ci) % b.width, int(ci) / b.width
}

func (b Board) Groups() []Group {
	return append(b.blocks, append(b.cols, b.rows...)...)
}

func (b *Board) SetValue(depth, reason string, x, y, val int) error {
	fmt.Printf("%s%s: (%d,%d) -> %d\n", depth, reason, x, y, val)
	ci := b.CellIndex(x, y)

	if !b.Cell(ci).CanTake(val) {
		x, y := b.IndexToCoords(ci)
		return fmt.Errorf("tried to set value on %d not legal at (%d,%d)", val, x, y)
	}

	if err := b.Cell(ci).SetValue(val); err != nil {
		return err
	}

	for _, g := range b.Groups() {
		if g.Contains(ci) {
			for _, i := range g.Indices() {
				if b.Cell(i).CanTake(val) {
					x, y := b.IndexToCoords(i)
					b.ProhibitValue(depth+" ", "excluding because of set value", x, y, val)
				}
			}
		}
	}
	return nil
}

func (b *Board) ProhibitValue(depth, reason string, x, y, val int) error {
	cell := b.CellAt(x, y)
	if !cell.CanTake(val) {
		return nil
	}

	fmt.Printf("%s%s: (%d,%d) cannot be %d\n", depth, reason, x, y, val)
	cell.Prohibit(val)
	if remaining := cell.Possibilities(b.height); len(remaining) == 1 {
		return b.SetValue(depth+" ", "only one left after prohibition", x, y, remaining[0])
	}

	return nil
}

func (b *Board) String() string {
	var buf bytes.Buffer

	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			cell := b.CellAt(x, y)
			if val, ok := cell.GetValue(); ok {
				fmt.Fprintf(&buf, "%d ", val)
			} else {
				fmt.Fprintf(&buf, "- ")
			}
			if (x+1)%b.blockWidth == 0 {
				fmt.Fprintf(&buf, "\t")
			}
		}
		if (y+1)%b.blockHeight == 0 {
			fmt.Fprintf(&buf, "\n")
		}
		fmt.Fprintf(&buf, "\n")
	}
	return buf.String()
}

func NewBoardFromBuffer(blockWidth, blockHeight, blockCountHoriz, blockCountVert int, input io.Reader) (*Board, error) {
	scanner := bufio.NewScanner(input)
	board := NewBoard(blockWidth, blockHeight, blockCountHoriz, blockCountVert)
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

func (b *Board) Solve() error {
	for {
		nakedGroups, err := b.solveNakedGroups()
		if err != nil {
			return err
		}
		hiddenSingles, err := b.solveHiddenSingles()
		if err != nil {
			return err
		}

		blockIntersects, err := b.solveBlockGroupIntersections()
		if err != nil {
			return err
		}

		if !nakedGroups && !hiddenSingles && !blockIntersects {
			break
		}
	}

	if b.IsSolved() {
		fmt.Printf("\nsolved!\n")
	} else {
		fmt.Printf("\ndidn't solve!\n")
		fmt.Println(b.Unsolved())
	}

	fmt.Println(b.String())
	return nil
}

func (b *Board) IsSolved() bool {
	for _, c := range b.cells {
		if _, set := c.GetValue(); !set {
			return false
		}
	}
	return true
}

func (b *Board) Unfilled() Group {
	var cells []CellIndex
	for ci, c := range b.cells {
		if _, set := c.GetValue(); !set {
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

func (b *Board) solveHiddenSingles() (bool, error) {
	var progress bool
	// iterate over all the groups
	for _, g := range b.Groups() {
		// make a map that collects the cells within the group that take a specific value
		m := make(map[int][]*Cell)
		for _, c := range g.Cells(b) {
			if c.Filled() {
				continue
			}
			for _, val := range c.Possibilities(b.height) {
				m[val] = append(m[val], c)
			}
		}
		for val, cells := range m {
			if len(cells) == 1 {
				x, y := b.Coords(cells[0])
				cell := b.CellAt(x, y)
				if !cell.Filled() {
					if err := b.SetValue("", "value can only appear in cell", x, y, val); err != nil {
						return false, fmt.Errorf("solveHiddenSingles: %w", err)
					}
					progress = true
				}
			}
		}
	}
	return progress, nil
}

func (b *Board) solveNakedGroups() (bool, error) {
	progress := false

	for _, g := range b.Groups() {
		unfilled := g.Unfilled(b)
		if unfilled.Len() == 0 {
			continue
		}
		unfilled.GenerateCombinations(func(combo Group) error {
			possible := combo.Possibilities(b)
			if len(possible) == combo.Len() {
				for _, ci := range unfilled.Indices() {
					if !combo.Contains(ci) {
						x, y := b.IndexToCoords(ci)
						for _, val := range possible {
							if b.Cell(ci).CanTake(val) {
								if err := b.ProhibitValue("", "naked group", x, y, val); err != nil {
									return err
								}
								progress = true
							}
						}
					}
				}
			}
			return nil
		})
	}
	return progress, nil
}

// if a number X can only appear in N cells of block A, and those N cells also appear in some other
// group B, then we can prohibit the number X from all the other cells of group B.
func (b *Board) solveBlockGroupIntersections() (bool, error) {
	var progress bool

	for _, block := range b.blocks {
		for _, val := range block.Possibilities(b) {
			set := block.CanTake(val, b)
			for _, other := range append(b.rows, b.cols...) {
				if set.ContainedBy(other) {
					for _, ci := range other.Indices() {
						if !set.Contains(ci) {
							x, y := b.IndexToCoords(ci)
							if b.Cell(ci).CanTake(val) {
								if err := b.ProhibitValue("", "block group intersection", x, y, val); err != nil {
									return false, err
								}
								progress = true
							}
						}
					}
				}
			}
		}
	}
	return progress, nil
}

func (b *Board) Unsolved() string {
	var buf bytes.Buffer

	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			c := b.CellAt(x, y)
			if c.Filled() {
				continue
			}
			fmt.Fprintf(&buf, "(%d,%d): %+v\n", x, y, c.Possibilities(b.height))
		}
	}
	return buf.String()
}

func (b *Board) Validate() {
	for _, g := range append(b.blocks, append(b.rows, b.cols...)...) {
		seen := make(map[int]struct{})
		for _, c := range g.Cells(b) {
			if v, set := c.GetValue(); set {
				if _, found := seen[v]; found {
					panic(fmt.Errorf("invalid board; duplicate %d found", v))
				}
			}
		}
	}
}
