package main

type Group []*Cell

func NewColumnGroup(board *Board, colIndex int) Group {
	var g Group

	for y := 0; y < board.height; y++ {
		g = append(g, board.Cell(colIndex, y))
	}
	return g
}

func NewRowGroup(board *Board, rowIndex int) Group {
	var g Group

	for x := 0; x < board.width; x++ {
		g = append(g, board.Cell(x, rowIndex))
	}
	return g
}

func NewBlockGroup(board *Board, blockX, blockY, blockWidth, blockHeight int) Group {
	var g Group

	for x := 0; x < blockWidth; x++ {
		for y := 0; y < blockHeight; y++ {
			g = append(g, board.Cell(blockX*blockWidth+x, blockY*blockHeight+y))
		}
	}
	return g
}

func (g Group) Prohibit(val int) {
	for _, c := range g {
		if !c.Filled() {
			c.Prohibit(val)
		}
	}
}

func (g Group) Contains(c *Cell) bool {
	for _, cell := range g {
		if cell == c {
			return true
		}
	}
	return false
}

func (g Group) Unfilled() Group {
	count := 0

	for _, cell := range g {
		if !cell.Filled() {
			count++
		}
	}

	unfilled := make([]*Cell, 0, count)

	for _, cell := range g {
		if !cell.Filled() {
			unfilled = append(unfilled, cell)
		}
	}

	return Group(unfilled)
}

func (g Group) Possibilities(count int) []int {
	var result []int
	mask := (1 << count) - 1
	var bits int

	for _, c := range g {
		bits = bits | (c.not ^ mask)
	}

	for i := 1; bits > 0; i, bits = i+1, bits>>1 {
		if bits&1 == 1 {
			result = append(result, i)
		}
	}
	return result
}

func (g Group) GenerateCombinations(callback func(combo Group) error) error {
	count := len(g)
	max := 1 << count

	for i := 0; i < max; i++ {
		var list []*Cell

		for j, mask := 0, i; j < len(g) && mask > 0; j, mask = j+1, mask>>1 {
			if mask&1 == 1 {
				list = append(list, g[j])
			}
		}

		if err := callback(Group(list)); err != nil {
			return err
		}
	}
	return nil
}
