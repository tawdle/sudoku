package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type Board struct {
	cells       []Cell
	width       int
	height      int
	blockWidth  int
	blockHeight int
	groups      []Group
}

func NewBoard(blockWidth, blockHeight, blockCountHoriz, blockCountVert int) *Board {
	// first, create all our cells
	board := &Board{}
	board.width = blockWidth * blockCountHoriz
	board.height = blockHeight * blockCountVert
	board.blockWidth = blockWidth
	board.blockHeight = blockHeight
	board.cells = make([]Cell, board.width*board.height)

	for x := 0; x < blockCountHoriz; x++ {
		for y := 0; y < blockCountVert; y++ {
			board.groups = append(board.groups, NewBlockGroup(board, x, y, blockWidth, blockHeight))
		}
	}

	for x := 0; x < board.width; x++ {
		board.groups = append(board.groups, NewColumnGroup(board, x))
	}

	for y := 0; y < board.height; y++ {
		board.groups = append(board.groups, NewRowGroup(board, y))
	}

	return board
}

func (b *Board) Cell(x, y int) *Cell {
	return &b.cells[y*b.width+x]
}

func (b *Board) Coords(c *Cell) (x int, y int) {
	for i := range b.cells {
		if c == &b.cells[i] {
			return i % b.width, i / b.width
		}
	}
	panic(fmt.Errorf("couldn't find cell %+v", c))
}

func (b *Board) SetValue(x, y, val int) error {
	fmt.Printf("setting value for cell (%d,%d) -> %d\n", x, y, val)
	cell := b.Cell(x, y)

	if err := cell.SetValue(val); err != nil {
		return err
	}

	for _, g := range b.groups {
		if g.Contains(cell) {
			g.Prohibit(val)
		}
	}
	return nil
}

func (b *Board) SetCellValue(c *Cell, val int) error {
	x, y := b.Coords(c)
	return b.SetValue(x, y, val)
}

func (b *Board) String() string {
	var buf bytes.Buffer

	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			cell := b.Cell(x, y)
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
			if err := board.SetValue(x, y, int(val)); err != nil {
				return nil, err
			}
		}
		y++
	}
	return board, nil
}

func (b *Board) Solve() error {
	for {
		singles, err := b.solveNakedSingles()
		if err != nil {
			return err
		}
		pairs, err := b.solveNakedGroups()
		if err != nil {
			return err
		}
		hiddenSingles, err := b.solveHiddenSingles()
		if err != nil {
			return err
		}

		if !singles && !pairs && !hiddenSingles {
			break
		}
	}

	if b.IsSolved() {
		fmt.Printf("solved!\n")
	} else {
		fmt.Printf("didn't solve!\n")
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

func (b *Board) solveNakedSingles() (bool, error) {
	progress := false

	for x := 0; x < b.width; x++ {
		for y := 0; y < b.height; y++ {
			c := b.Cell(x, y)
			if _, set := c.GetValue(); set {
				continue
			}
			if possible := c.Possibilities(b.height); len(possible) == 1 {
				fmt.Printf("only one possibility for cell (%d,%d):: %d\n", x, y, possible[0])
				if err := b.SetValue(x, y, possible[0]); err != nil {
					return false, fmt.Errorf("solveNakedSingles: %w", err)
				}
				progress = true
			}
		}
	}
	return progress, nil
}

func within(c *Cell, list []*Cell) bool {
	for _, member := range list {
		if c == member {
			return true
		}
	}
	return false
}

func (b *Board) solveHiddenSingles() (bool, error) {
	var progress bool
	// iterate over all the groups
	for _, g := range b.groups {
		// make a map that collects that cells within the group that take a specific value
		m := make(map[int][]*Cell)
		for _, c := range g {
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
				fmt.Printf("value %d can only appear in cell (%d,%d)\n", val, x, y)
				if err := b.SetCellValue(cells[0], val); err != nil {
					return false, fmt.Errorf("solveHiddenSingles: %w", err)
				}
				progress = true
			}
		}
	}
	return progress, nil
}

func (b *Board) solveNakedGroups() (bool, error) {
	progress := false
	// for each group
	for _, g := range b.groups {
		// construct a map that collects together cells that have
		// identical possibilities
		m := make(map[int][]*Cell)
		for _, c := range g {
			if !c.Filled() {
				m[c.not] = append(m[c.not], c)
			}
		}
		// iterate over the map, looking for sets of N cells with N possibilities
		for _, cells := range m {
			possibilities := cells[0].Possibilities(b.height)
			// did we find a such a set?
			if len(possibilities) == len(cells) {
				fmt.Printf("found a naked group with %+v\n", possibilities)
				// iterate over the cells in the group
				for _, c := range g {
					// skip over cells that are within our set
					if within(c, cells) {
						continue
					}
					for _, val := range possibilities {
						if c.CanTake(val) {
							progress = true
							x, y := b.Coords(c)
							fmt.Printf("prohibiting %d from (%d,%d)\n", val, x, y)
							if err := c.Prohibit(val); err != nil {
								return false, err
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
			c := b.Cell(x, y)
			if c.Filled() {
				continue
			}
			fmt.Fprintf(&buf, "(%d,%d): %+v\n", x, y, c.Possibilities(b.height))
		}
	}
	return buf.String()
}
