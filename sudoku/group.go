package main

type Group struct {
	board *Board
	cells []CellIndex // list of cells by index
}

func NewGroup(board *Board, ci []CellIndex) Group {
	return Group{
		board: board,
		cells: ci,
	}
}

func NewColumnGroup(board *Board, colIndex int) Group {
	cells := make([]CellIndex, 0, board.height)

	for y := 0; y < board.height; y++ {
		cells = append(cells, board.CellIndex(colIndex, y))
	}
	return NewGroup(board, cells)
}

func NewRowGroup(board *Board, rowIndex int) Group {
	cells := make([]CellIndex, 0, board.width)

	for x := 0; x < board.width; x++ {
		cells = append(cells, board.CellIndex(x, rowIndex))
	}
	return NewGroup(board, cells)
}

func NewBlockGroup(board *Board, blockX, blockY, blockWidth, blockHeight int) Group {
	cells := make([]CellIndex, 0, blockWidth*blockHeight)

	for x := 0; x < blockWidth; x++ {
		for y := 0; y < blockHeight; y++ {
			cells = append(cells, board.CellIndex(blockX*blockWidth+x, blockY*blockHeight+y))
		}
	}
	return NewGroup(board, cells)
}

func (g Group) Len() int {
	return len(g.cells)
}

func (g Group) Indices() []CellIndex {
	return g.cells
}

func (g Group) Cells() []*Cell {
	cells := make([]*Cell, 0, len(g.cells))

	for _, ci := range g.cells {
		cells = append(cells, g.board.Cell(ci))
	}
	return cells
}

func (g Group) Prohibit(val int) {
	for _, c := range g.Cells() {
		if !c.Filled() {
			c.Prohibit(val)
		}
	}
}

func (g Group) Contains(ci CellIndex) bool {
	for _, cell := range g.cells {
		if cell == ci {
			return true
		}
	}
	return false
}

func (g Group) Unfilled() Group {
	cells := make([]CellIndex, 0, len(g.cells))

	for _, ci := range g.cells {
		if !g.board.Cell(ci).Filled() {
			cells = append(cells, ci)
		}
	}

	return NewGroup(g.board, cells)
}

func (g Group) Possibilities() []int {
	var result []int
	mask := (1 << g.board.height) - 1
	var bits int

	for _, c := range g.Cells() {
		bits = bits | (c.not ^ mask)
	}

	for i := 1; bits > 0; i, bits = i+1, bits>>1 {
		if bits&1 == 1 {
			result = append(result, i)
		}
	}
	return result
}

func (g Group) CanTake(val int) Group {
	cells := make([]CellIndex, 0, len(g.cells))

	for _, ci := range g.cells {
		if g.board.Cell(ci).CanTake(val) {
			cells = append(cells, ci)
		}
	}
	return NewGroup(g.board, cells)
}

func (g Group) ContainedBy(other Group) bool {
	for _, c := range g.cells {
		if !other.Contains(c) {
			return false
		}
	}
	return true
}

func (g Group) GenerateCombinations(callback func(combo Group) error) error {
	count := len(g.cells)
	max := 1 << count

	for i := 1; i < max; i++ {
		var list []CellIndex

		for j, mask := 0, i; j < len(g.cells) && mask > 0; j, mask = j+1, mask>>1 {
			if mask&1 == 1 {
				list = append(list, g.cells[j])
			}
		}

		if err := callback(NewGroup(g.board, list)); err != nil {
			return err
		}
	}
	return nil
}

func (g Group) Intersection(other Group) Group {
	cells := make([]CellIndex, 0, len(g.cells))

	for _, c := range g.cells {
		if other.Contains(c) {
			cells = append(cells, c)
		}
	}

	return NewGroup(g.board, cells)
}

func (g Group) Intersects(other Group) bool {
	for _, c := range g.cells {
		if other.Contains(c) {
			return true
		}
	}

	return false
}
