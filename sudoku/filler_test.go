package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnset(t *testing.T) {
	assert := assert.New(t)
	board, err := Fill(NewBoard(2, 2))
	assert.NoError(err)
	board.UnsetValue(10)
	numSet := board.Filled().Len()
	assert.Equal(15, numSet)
	for i := 0; i < 16; i++ {
		cell := board.Cell(CellIndex(i))
		fmt.Printf("%d: [%p] %s\n", i, cell, cell)
	}
}

func TestFiller(t *testing.T) {
	assert := assert.New(t)
	solution, err := Fill(NewBoard(3, 3))
	assert.NoError(err)
	assert.True(solution.IsSolved())

	fmt.Println("Generated a board...")
	fmt.Println(solution.String())
}

func TestMakePuzzle(t *testing.T) {
	assert := assert.New(t)
	board := NewBoard(3, 3)
	board.quiet = true
	solution, err := Fill(board)
	if assert.NoError(err) {
		fmt.Println("generated a full board:")
		fmt.Println(solution)
		if assert.NoError(err) {
			puzzle, err := MakePuzzle(solution, 24)
			if assert.NoError(err) && assert.NoError(puzzle.IsValid()) {
				fmt.Println("generated a puzzle...")
				fmt.Println(puzzle)
				fmt.Println(puzzle.Unsolved())
			}
		}
	}
}
