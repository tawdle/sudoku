package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSolver(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		title  string
		puzzle string
	}{
		{
			title: "NYT Easy",
			puzzle: `6437-8--5
		 5-8139
		 ----6--32
		 35--9-1-4
		 -6-4175
		 -9----2
		 1--3--45
		 --5--1-68
		 98---5-2`,
		},
		{
			title: "NYT Medium",
			puzzle: `---5--4
			9--32--5
			---18--9
			--57--36
			---2-6
			--14---7
			-43---7-2
			-5
			--------1`,
		},
		{
			title: "NYT Hard",
			puzzle: `-7---21
			----9-35
			--5--4
			7
			6--9--2-4
			2-18-3-76

			---16
			382`,
		},
	}

	for _, c := range cases {
		board, err := NewBoardFromBuffer(3, 3, 3, 3, strings.NewReader(c.puzzle))
		assert.NoError(err)
		fmt.Printf("trying to solve %s...\n", c.title)
		err = board.Solve()
		assert.NoError(err)
		assert.True(board.IsSolved())
	}
}
