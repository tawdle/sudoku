package main

import (
	"fmt"
	"math/rand"
)

// Given a board in any state, returns a new board with all of its cells filled (or error)
func Fill(board *Board) (*Board, error) {
	// The basic idea here is a hyrbid of brute-force and heuristic.
	// At each step, we randomly pick an open cell, and iterate over each
	// of its candidates, trying each one in turn. After filling a cell
	// with a candidate, we attempt to solve the puzzle heuristically. This
	// leads to three possible outcomes:
	// 1. An error -- there's an inherent conflict; the candidate we picked leads
	// to a puzzle that cannot be solved.
	// 2. An unsolved puzzle: the candidate itself is okay, but the solver cannot
	// generate a unique solution.
	// 3. A solved puzzle: the solver was able to complete the puzzle with a unique
	// solution.
	// In the first case, we simply backtrack and try the next candidate.
	// In the second case, we pick another random cell and continue.
	// In the third case, we've succeeded, so we return the filled board.

	var fillNextCell func(*Board) (*Board, error)
	var tryCandidate func(*Board, CellIndex, int) (*Board, error)

	fillNextCell = func(board *Board) (*Board, error) {
		// Pick an unfilled cell randomly
		ci := board.Unfilled().PickOne()
		candidates := board.Cell(ci).Possibilities(board.Size())
		rand.Shuffle(len(candidates), func(x, y int) { candidates[x], candidates[y] = candidates[y], candidates[x] })
		for _, candidate := range candidates {
			if nb, err := tryCandidate(board.Duplicate(), ci, candidate); err == nil {
				return nb, nil
			}
		}
		// we tried all candidates for cell but none succeeded
		return nil, fmt.Errorf("no candidate worked for cell %d", int(ci))
	}

	tryCandidate = func(board *Board, ci CellIndex, value int) (*Board, error) {
		if err := board.IsValid(); err != nil {
			return nil, err
		}
		x, y := board.IndexToCoords(ci)
		if err := board.SetValue("", "random pick", x, y, value); err != nil {
			return nil, err
		}
		// we were able to set a value; now try to use solver to fill in some more
		if err := board.Solve(); err != nil {
			return nil, err
		}
		if board.IsSolved() {
			return board, nil
		}
		// still not solved; try filling more values randomly
		return fillNextCell(board)
	}

	return fillNextCell(board.Duplicate())

}

func (b *Board) Solve() error {
	for {
		nakedGroups, err := b.solveNakedGroups()
		if err != nil {
			return err
		}
		hiddenSingles, err := b.solveHiddenSingles()
		if err != nil {
			return err
		}

		blockIntersects, err := b.solveBlockGroupIntersections()
		if err != nil {
			return err
		}

		if !nakedGroups && !hiddenSingles && !blockIntersects {
			break
		}
	}

	if b.IsSolved() {
		if err := b.IsValid(); err == nil {
			if !b.quiet {
				fmt.Printf("\nsolved!\n")
			}
		} else {
			if !b.quiet {
				fmt.Printf("solved but solution is invalid! %s\n", err)
			}
		}
	} else {
		if !b.quiet {
			fmt.Printf("\ndidn't solve!\n")
			fmt.Println(b.Unsolved())
		}
	}

	if !b.quiet {
		fmt.Println(b.String())
	}

	return nil
}

func (b *Board) solveHiddenSingles() (bool, error) {
	var progress bool
	// iterate over all the groups
	for _, g := range b.Groups() {
		// make a map that collects the cells within the group that take a specific value
		m := make(map[int][]CellIndex)
		for _, ci := range g.Indices() {
			c := b.Cell(ci)
			if c.Filled() {
				continue
			}
			for _, val := range c.Possibilities(b.Size()) {
				m[val] = append(m[val], ci)
			}
		}
		for val, cis := range m {
			if len(cis) == 1 {
				x, y := b.IndexToCoords(cis[0])
				cell := b.Cell(cis[0])
				if !cell.Filled() {
					if err := b.SetValue("", "value can only appear in cell", x, y, val); err != nil {
						return false, fmt.Errorf("solveHiddenSingles: %w", err)
					}
					progress = true
				}
			}
		}
	}
	return progress, nil
}

func (b *Board) solveNakedGroups() (bool, error) {
	progress := false

	for _, g := range b.Groups() {
		unfilled := g.Unfilled(b)
		if unfilled.Len() == 0 {
			continue
		}
		err := unfilled.GenerateCombinations(func(combo *Group) error {
			combo = combo.Unfilled(b) // filter again here because cells may have been filled since we started
			possible := combo.Possibilities(b)
			if len(possible) == combo.Len() {
				for _, ci := range unfilled.Indices() {
					if !combo.Contains(ci) {
						x, y := b.IndexToCoords(ci)
						for _, val := range possible {
							if b.Cell(ci).CanTake(val) {
								if err := b.ProhibitValue("", fmt.Sprintf("naked group %v", possible), x, y, val); err != nil {
									return err
								}
								progress = true
							}
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return false, err
		}
	}
	return progress, nil
}

// if a number X can only appear in N cells of block A, and those N cells also appear in some other
// group B, then we can prohibit the number X from all the other cells of group B.
func (b *Board) solveBlockGroupIntersections() (bool, error) {
	var progress bool

	for _, block := range b.spec.blocks {
		for _, val := range block.Possibilities(b) {
			set := block.CanTake(val, b)
			if set.Len() < 2 {
				continue
			}
			for _, other := range append(b.spec.rows, b.spec.cols...) {
				if set.ContainedBy(other) {
					for _, ci := range other.Indices() {
						if !set.Contains(ci) {
							if b.Cell(ci).CanTake(val) {
								x, y := b.IndexToCoords(ci)
								if err := b.ProhibitValue("", "block group intersection", x, y, val); err != nil {
									return false, err
								}
								progress = true
							}
						}
					}
				}
			}
		}
	}
	return progress, nil
}

// MakePuzzle generates a puzzle whose solution matches the provided board,
// which must be a complete and valid board.  The original board remains
// untocued.  When we reach "target" number of filled cells in our puzzle, we
// are done.
func MakePuzzle(b *Board, target int) (*Board, error) {
	if !b.IsSolved() {
		return nil, fmt.Errorf("provided board is not complete")
	}
	if err := b.IsValid(); err != nil {
		return nil, fmt.Errorf("provided board is not valid: %e", err)
	}

	var removeOne func(*Board) (*Board, error)

	removeOne = func(board *Board) (*Board, error) {
		// build a randomized list of filled cells
		filled := board.Filled().Shuffled()
		if !b.quiet {
			fmt.Printf("removeOne: board has %d filled cells\n", filled.Len())
		}

		for _, ci := range filled.Indices() {
			puzzle := board.Duplicate()
			//			val, _ := b.Cell(ci).GetValue()
			//			x, y := b.IndexToCoords(ci)
			//			fmt.Printf("removing value %d at (%d,%d)\n", val, x, y)
			puzzle.UnsetValue(ci)
			fmt.Printf(".")

			nb := puzzle.Duplicate()
			err := nb.Solve()
			// an error means we've hit a dead-end and need to backtrack
			// (not sure whether this can actually happen given that we
			// are starting with a solved board and working backwards,
			// but better safe than sorry)
			if err != nil {
				fmt.Printf("\nbacktracking: solver returned error: %s\n", err)
				continue
			}

			// if we weren't able to solve, then we have gone too far and there
			// isn't a unique solution, so we also need to backtrack
			if !nb.IsSolved() {
				//				fmt.Println("solution was ambiguous; trying next cell")
				continue
			}

			// We have a solveable puzzle; is it small enough yet?
			if puzzle.Filled().Len() <= target {
				fmt.Println("got a puzzle!")
				return puzzle, nil
			}

			// our path is good, but we're not done yet
			//			fmt.Printf("have a puzzle with %d filled\n", puzzle.Filled().Len())
			if finalPuzzle, err := removeOne(puzzle); err == nil {
				return finalPuzzle, nil
			}
		}
		return board, fmt.Errorf("couldn't generate a puzzle with requested target; only got to %d", board.Unfilled().Len())
	}

	return removeOne(b)
}
