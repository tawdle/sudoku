package main

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
)

type Group struct {
	cells []CellIndex // list of cells by index
}

func NewGroup(ci []CellIndex) *Group {
	return &Group{
		cells: ci,
	}
}

func NewColumnGroup(board *Board, colIndex int) *Group {
	cells := make([]CellIndex, 0, board.Size())

	for y := 0; y < board.Size(); y++ {
		cells = append(cells, board.CellIndex(colIndex, y))
	}
	return NewGroup(cells)
}

func NewRowGroup(board *Board, rowIndex int) *Group {
	cells := make([]CellIndex, 0, board.Size())

	for x := 0; x < board.Size(); x++ {
		cells = append(cells, board.CellIndex(x, rowIndex))
	}
	return NewGroup(cells)
}

func NewBlockGroup(board *Board, blockX, blockY, blockWidth, blockHeight int) *Group {
	cells := make([]CellIndex, 0, blockWidth*blockHeight)

	for x := 0; x < blockWidth; x++ {
		for y := 0; y < blockHeight; y++ {
			cells = append(cells, board.CellIndex(blockX*blockWidth+x, blockY*blockHeight+y))
		}
	}
	return NewGroup(cells)
}

func (g *Group) Clone() *Group {
	return NewGroup(slices.Clone(g.cells))
}

func (g *Group) Len() int {
	return len(g.cells)
}

func (g *Group) First() CellIndex {
	if len(g.cells) < 1 {
		panic(fmt.Errorf("called First() on empty group"))
	}
	return g.cells[0]
}

func (g *Group) Last() CellIndex {
	if len(g.cells) < 1 {
		panic(fmt.Errorf("called Last() on empty group"))
	}
	return g.cells[len(g.cells)-1]
}

// Minus() returns a new Group with the specified
// Cellindex(es) removed
func (g *Group) Minus(cis ...CellIndex) *Group {
	cells := make([]CellIndex, 0, len(g.cells))
	for _, source := range g.cells {
		found := false
		for _, remove := range cis {
			if source == remove {
				found = true
				break
			}
		}
		if !found {
			cells = append(cells, source)
		}
	}
	return NewGroup(cells)
}

func (g *Group) Indices() []CellIndex {
	return g.cells
}

func (g *Group) Cells(b *Board) []*Cell {
	cells := make([]*Cell, 0, len(g.cells))

	for _, ci := range g.cells {
		cells = append(cells, b.Cell(ci))
	}
	return cells
}

func (g *Group) Prohibit(val int, b *Board) {
	for _, c := range g.Cells(b) {
		if !c.Filled() {
			c.Prohibit(val)
		}
	}
}

func (g *Group) Contains(ci CellIndex) bool {
	for _, cell := range g.cells {
		if cell == ci {
			return true
		}
	}
	return false
}

func (g *Group) FindSetValue(value int, b *Board) (CellIndex, bool) {
	for _, ci := range g.cells {
		if val, set := b.Cell(ci).GetValue(); set && val == value {
			return ci, true
		}
	}
	return 0, false
}

func (g *Group) Unfilled(b *Board) *Group {
	cells := make([]CellIndex, 0, len(g.cells))

	for _, ci := range g.cells {
		if !b.Cell(ci).Filled() {
			cells = append(cells, ci)
		}
	}

	return NewGroup(cells)
}

func (g *Group) Possibilities(b *Board) []int {
	var result []int
	mask := (1 << b.Size()) - 1
	var bits int

	for _, c := range g.Cells(b) {
		if !c.IsSet() {
			bits = bits | (c.not ^ mask)
		}
	}

	for i := 1; bits > 0; i, bits = i+1, bits>>1 {
		if bits&1 == 1 {
			result = append(result, i)
		}
	}
	return result
}

func (g *Group) CanTake(val int, b *Board) *Group {
	cells := make([]CellIndex, 0, len(g.cells))

	for _, ci := range g.cells {
		if b.Cell(ci).CanTake(val) {
			cells = append(cells, ci)
		}
	}
	return NewGroup(cells)
}

func (g *Group) ContainedBy(other *Group) bool {
	for _, c := range g.cells {
		if !other.Contains(c) {
			return false
		}
	}
	return true
}

func (g *Group) GenerateCombinations(callback func(combo *Group) error) error {
	count := len(g.cells)
	max := 1 << count

	for i := 1; i < max; i++ {
		var list []CellIndex

		for j, mask := 0, i; j < len(g.cells) && mask > 0; j, mask = j+1, mask>>1 {
			if mask&1 == 1 {
				list = append(list, g.cells[j])
			}
		}

		if err := callback(NewGroup(list)); err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) Intersection(other *Group) *Group {
	cells := make([]CellIndex, 0, len(g.cells))

	for _, c := range g.cells {
		if other.Contains(c) {
			cells = append(cells, c)
		}
	}

	return NewGroup(cells)
}

func (g *Group) Intersects(other Group) bool {
	for _, c := range g.cells {
		if other.Contains(c) {
			return true
		}
	}

	return false
}

func (g *Group) SetValues(b *Board) []int {
	seen := make(map[int]struct{})

	for _, ci := range g.cells {
		if val, set := b.Cell(ci).GetValue(); set {
			seen[val] = struct{}{}
		}
	}
	var result []int

	for val, _ := range seen {
		result = append(result, val)
	}
	return result
}

func (g *Group) PickOne() CellIndex {
	if len(g.cells) == 0 {
		panic(fmt.Errorf("can't pick one from an empty group"))
	}
	return g.cells[rand.Intn(len(g.cells))]
}

// Shuffled() returns a new group containing the same cells
// of the current group, but in a random order.
// Leaves original group unmodified.
func (g *Group) Shuffled() *Group {
	cells := slices.Clone(g.cells)
	rand.Shuffle(len(cells), func(x, y int) {
		cells[x], cells[y] = cells[y], cells[x]
	})
	return NewGroup(cells)
}

// Reversed returns a new group containing the same
// of the current group, but in reverse order.
// Leaves original group unmodified.
func (g *Group) Reversed() *Group {
	cells := slices.Clone(g.cells)
	slices.Reverse(cells)
	return NewGroup(cells)
}

func (g *Group) Append(cells ...CellIndex) *Group {
	g.cells = append(g.cells, cells...)
	return g
}

func (g *Group) MapCandidatesToCells(b *Board) map[int]*Group {
	m := make(map[int]*Group)

	for _, c := range g.cells {
		for _, candidate := range b.Cell(c).Candidates(b.Size()) {
			if ng, exists := m[candidate]; !exists {
				m[candidate] = NewGroup([]CellIndex{c})
			} else {
				ng.Append(c)
			}
		}
	}
	return m
}

func (g *Group) String(b *Board) string {
	var result strings.Builder
	result.WriteString("[ ")
	for _, ci := range g.cells {
		x, y, _ := b.IndexToCoords(ci)
		fmt.Fprintf(&result, "(%d,%d) ", x, y)
	}
	result.WriteString("]")
	return result.String()
}
