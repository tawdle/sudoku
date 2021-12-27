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
		{
			title: "NYT Medium Puzzle 12/23",
			puzzle: `5-8---713
			-6------5
			3
			--2--9--1
			---64
			-3-81-9
			12-----87
			---4--6
			4-5`,
		},
		{
			title: "NYT Hard Puzzle 12/23",
			puzzle: `-75-3
			--1----3
			3-----798

			-----5-1
			---27-48
			---98
			-47--63
			-6---2`,
		},
		{
			title: "NYT Hard 12/24",
			puzzle: `-5-91
			--6-3
			--17--3-4
			--3--81
			2---93--5
			-4
			--------9
			----5-421
			-----2-7-`,
		},
		{
			title: "NYT Hard 12/25",
			puzzle: `-8-2---9
			-6-----73
			--1--82
			------9
			5---6---4
			1-79---3
			--------2
			824-7
			---1----8`,
		},
		{
			title: "NYT Hard 12/27/2021",
			puzzle: `--1--4
			48--69
			9-3----6

			------3-5
			1----7-84
			-3----2
			-7-9--5

			
			--97-8`,
		},
	}

	for _, c := range cases {
		fmt.Printf("trying to solve %s...\n", c.title)
		board, err := NewBoardFromBuffer(3, 3, 3, 3, strings.NewReader(c.puzzle))
		assert.NoError(err)
		err = board.Solve()
		assert.NoError(err)
		assert.True(board.IsSolved())
	}
}
