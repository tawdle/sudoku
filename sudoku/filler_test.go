package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFiller(t *testing.T) {
	assert := assert.New(t)
	solution, err := Fill(NewBoard(3, 3))
	assert.NoError(err)
	assert.True(solution.IsSolved())

	fmt.Println("Generated a board...")
	fmt.Println(solution.String())
}
