package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExclusions(t *testing.T) {
	assert := assert.New(t)

	board := NewBoard(2, 2)
	assert.NoError(board.SetValue("", "testing", 1, 1, 2))
	if !assert.Error(board.SetValue("", "testing", 3, 1, 2)) {
		fmt.Printf(board.Unsolved())
	}
}
