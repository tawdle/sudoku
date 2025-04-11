package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strings"
)

type CellIndex int

type BoardSpec struct {
	size        int
	blockWidth  int
	blockHeight int
	blocks      []*Group
	cols        []*Group
	rows        []*Group
}

type Board struct {
	cells []Cell
	spec  *BoardSpec
	quiet bool
}

// Creates an empty sudoku board with blocks of the specified
// dimension. Note that these dimensions determine the overall
// layout of the board.
func NewBoard(blockWidth, blockHeight int) *Board {
	// first, create all our cells
	board := &Board{}
	spec := &BoardSpec{}
	board.spec = spec

	spec.size = blockWidth * blockHeight
	spec.blockWidth = blockWidth
	spec.blockHeight = blockHeight
	board.cells = make([]Cell, spec.size*spec.size)

	var blocks, cols, rows []*Group

	for y := 0; y < blockWidth; y++ {
		for x := 0; x < blockHeight; x++ {
			blocks = append(blocks, NewBlockGroup(board, x, y, blockWidth, blockHeight))
		}
	}

	for x := 0; x < spec.size; x++ {
		cols = append(cols, NewColumnGroup(board, x))
	}

	for y := 0; y < spec.size; y++ {
		rows = append(rows, NewRowGroup(board, y))
	}

	spec.blocks = blocks
	spec.cols = cols
	spec.rows = rows

	return board
}

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

func (b *Board) Duplicate() *Board {
	var nb Board

	nb.cells = make([]Cell, len(b.cells))
	copy(nb.cells, b.cells)
	nb.spec = b.spec
	nb.quiet = b.quiet
	return &nb
}

func (b *Board) Size() int {
	return b.spec.size
}

func (b *Board) Cell(ci CellIndex) *Cell {
	return &b.cells[ci]
}

func (b *Board) CellAt(x, y int) *Cell {
	return &b.cells[b.CellIndex(x, y)]
}

func (b *Board) CellIndex(x, y int) CellIndex {
	return CellIndex(x + y*b.Size())
}

func (b *Board) Coords(c *Cell) (x int, y int) {
	for i := range b.cells {
		if c == &b.cells[i] {
			return b.IndexToCoords(CellIndex(i))
		}
	}
	panic(fmt.Errorf("couldn't find cell %+v", c))
}

func (b *Board) IndexToCoords(ci CellIndex) (x, y int) {
	return int(ci) % b.Size(), int(ci) / b.Size()
}

func (b Board) Groups() []*Group {
	return append(b.spec.blocks, append(b.spec.cols, b.spec.rows...)...)
}

func (b *Board) GroupsContaining(ci CellIndex) []*Group {
	result := make([]*Group, 0, 3)

	x, y := b.IndexToCoords(ci)

	result = append(result, b.spec.cols[x])
	result = append(result, b.spec.rows[y])
	result = append(result, b.spec.blocks[x/b.spec.blockWidth+y/b.spec.blockHeight*b.spec.blockHeight])
	return result
}

func (b *Board) SetValue(depth, reason string, x, y, val int) error {
	ci := b.CellIndex(x, y)

	// already set to the given value? just exit
	if v, set := b.Cell(ci).GetValue(); set && val == v {
		return nil
	}

	if !b.Cell(ci).CanTake(val) {
		x, y := b.IndexToCoords(ci)
		return fmt.Errorf("tried to set value on %d not legal at (%d,%d)", val, x, y)
	}

	if !b.quiet {
		fmt.Printf("%s%s: (%d,%d) -> %d\n", depth, reason, x, y, val)
	}
	if err := b.Cell(ci).SetValue(val); err != nil {
		return err
	}

	// In order to avoid corruption of the board, we need to handle
	// prohibition in two steps: first, we mark all the effected cells,
	// then, we deal with the fallout from any of those markings
	var marked []CellIndex
	for _, g := range b.GroupsContaining(ci) {
		if g.Contains(ci) {
			for _, i := range g.Indices() {
				if i != ci && b.Cell(i).CanTake(val) {
					x, y := b.IndexToCoords(i)
					if err := b.ProhibitValue(depth+" ", "excluding because of set value", x, y, val); err != nil {
						return err
					}
					marked = append(marked, ci)
				} else {
					// this won't impact anything about how the puzzle gets solved,
					// but it does ensure our candidate lists remain in sync
					b.Cell(i).Prohibit(val)
				}
			}
		}
	}

	for _, ci := range marked {
		candidates := b.Cell(ci).Possibilities(b.Size())
		if len(candidates) == 1 {
			x, y := b.IndexToCoords(ci)
			if err := b.SetValue(depth+" ", "only one left after prohibition", x, y, candidates[0]); err != nil {
				return err
			}
		}
	}
	return b.IsValid()
}

// UnsetValue clears the value in the specified cell then
// resets and recalculates prohibition/candidate lists each
// impacted cell.
func (b *Board) UnsetValue(ci CellIndex) {
	cell := b.Cell(ci)
	if _, isSet := cell.GetValue(); isSet {
		cell.value = 0
		for _, g := range b.GroupsContaining(ci) {
			for _, oci := range g.Indices() {
				b.Cell(oci).ClearProhibitions()
				for _, og := range b.GroupsContaining(oci) {
					for _, val := range og.SetValues(b) {
						b.Cell(oci).Prohibit(val)
					}
				}
			}
		}
	}
}

func (b *Board) ProhibitValue(depth, reason string, x, y, val int) error {
	cell := b.CellAt(x, y)
	if cell.CanTake(val) {
		if !b.quiet {
			fmt.Printf("%s%s: (%d,%d) cannot be %d\n", depth, reason, x, y, val)
		}
	}
	if err := cell.Prohibit(val); err != nil {
		return err
	}

	return nil
}

func (b *Board) String() string {
	var buf bytes.Buffer

	for y := 0; y < b.Size(); y++ {
		for x := 0; x < b.Size(); x++ {
			cell := b.CellAt(x, y)
			if val, ok := cell.GetValue(); ok {
				fmt.Fprintf(&buf, "%d ", val)
			} else {
				fmt.Fprintf(&buf, "- ")
			}
			if (x+1)%b.spec.blockWidth == 0 {
				fmt.Fprintf(&buf, "\t")
			}
		}
		if (y+1)%b.spec.blockHeight == 0 {
			fmt.Fprintf(&buf, "\n")
		}
		fmt.Fprintf(&buf, "\n")
	}
	return buf.String()
}

func NewBoardFromBuffer(blockWidth, blockHeight int, input io.Reader) (*Board, error) {
	scanner := bufio.NewScanner(input)
	board := NewBoard(blockWidth, blockHeight)
	y := 0

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		for x, c := range line {
			if c == '-' || c == ' ' {
				continue
			}
			val := c - '0'
			if err := board.SetValue("", "initial value", x, y, int(val)); err != nil {
				return nil, err
			}
		}
		y++
	}
	return board, nil
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

func (b *Board) IsSolved() bool {
	for _, c := range b.cells {
		if _, set := c.GetValue(); !set {
			return false
		}
	}
	return true
}

func (b *Board) Unfilled() *Group {
	var cells []CellIndex
	for ci, c := range b.cells {
		if _, set := c.GetValue(); !set {
			cells = append(cells, CellIndex(ci))
		}
	}
	return NewGroup(cells)
}

func (b *Board) Filled() *Group {
	var cells []CellIndex
	for ci, c := range b.cells {
		if _, set := c.GetValue(); set {
			cells = append(cells, CellIndex(ci))
		}
	}
	return NewGroup(cells)
}

func within(ci int, list []int) bool {
	for _, member := range list {
		if ci == member {
			return true
		}
	}
	return false
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

func (b *Board) Unsolved() string {
	var buf bytes.Buffer

	for y := 0; y < b.Size(); y++ {
		for x := 0; x < b.Size(); x++ {
			c := b.CellAt(x, y)
			if c.Filled() {
				continue
			}
			fmt.Fprintf(&buf, "(%d,%d): %+v\n", x, y, c.Possibilities(b.Size()))
		}
	}
	return buf.String()
}

func (b *Board) IsValid() error {
	for _, g := range append(b.spec.blocks, append(b.spec.rows, b.spec.cols...)...) {
		seen := make(map[int]CellIndex)
		for _, ci := range g.Indices() {
			c := b.Cell(ci)
			if v, set := c.GetValue(); set {
				if _, found := seen[v]; found {
					x1, y1 := b.IndexToCoords(ci)
					x2, y2 := b.IndexToCoords(seen[v])
					if !b.quiet {
						fmt.Println("invalid board detected")
					}
					panic(fmt.Errorf("(%d,%d) and (%d,%d) are both %d", x1, y1, x2, y2, v))
				} else {
					seen[v] = ci
				}
			}
		}
	}

	for ci, c := range b.cells {
		if !c.IsSet() {
			if len(c.Possibilities(b.Size())) == 0 {
				if !b.quiet {
					fmt.Println("invalid board detected")
				}
				x, y := b.IndexToCoords(CellIndex(ci))
				return fmt.Errorf("(%d,%d) is unfilled and has no candidates", x, y)
			}
		}
	}

	for ci, c := range b.cells {
		if c.IsSet() {
			continue
		}
		for val := 1; val <= b.Size(); val++ {
			found := false
			var index CellIndex
			for _, g := range b.GroupsContaining(CellIndex(ci)) {
				if index, found = g.FindSetValue(val, b); found {
					break
				}
			}
			if found && c.CanTake(val) {
				x, y := b.IndexToCoords(CellIndex(ci))
				x2, y2 := b.IndexToCoords(index)
				panic(fmt.Errorf("(%d,%d) says it can take %d but (%d,%d)=%d", x, y, val, x2, y2, val))
			}

			/* Attenpting to validate the consistency of the prohibition list but
			this isn't correct: things can be prohibited for more complex reasons.

			If something isn't on the prohibition list, but should be, *that* is an error.


			if !found && !c.CanTake(val) {
				x, y := b.IndexToCoords(CellIndex(ci))
				panic(fmt.Errorf("(%d,%d) says it cannot take %d but it can", x, y, val))
			}
			*/
		}
	}

	return nil
}

func (b *Board) Validate() {
	if err := b.IsValid(); err != nil {
		panic(fmt.Errorf("invalid board: %e", err))
	}
}

// MakePuzzle generates a puzzle whose solution matches
// this board, which must be a complete and valid board.
// The original board remains untocued.
// When we reach "target" number of filled cells in our
// puzzle, we are done.
func (b *Board) MakePuzzle(target int) (*Board, error) {
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
