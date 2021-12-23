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
