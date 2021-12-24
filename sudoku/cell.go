package main

import "fmt"

type Cell struct {
	value int
	not   int // bitfield of prohibited values
}

func (c *Cell) SetValue(val int) error {
	if existingValue, set := c.GetValue(); set {
		if val != existingValue {
			return fmt.Errorf("attempted to set value on cell that has already been set")
		}
		return nil
	}

	if !c.CanTake(val) {
		return fmt.Errorf("attempted to set prohibited value %d", val)
	}
	c.value = val
	return nil
}

func (c *Cell) Prohibit(val int) error {
	if c.Filled() {
		return fmt.Errorf("tried to prohibit value in cell that already has one")
	}
	c.not |= (1 << (val - 1))
	return nil
}

func (c *Cell) CanTake(val int) bool {
	if c.Filled() {
		return false
	}
	return c.not&(1<<(val-1)) == 0
}

func (c *Cell) Filled() bool {
	return c.value != 0
}

func (c *Cell) GetValue() (val int, ok bool) {
	return c.value, c.value != 0
}

func (c *Cell) Possibilities(valueCount int) []int {
	if c.Filled() {
		return nil
	}

	var result []int

	mask := (1 << valueCount) - 1
	bits := c.not ^ mask
	count := 1

	for bits != 0 {
		if bits&1 != 0 {
			result = append(result, count)
		}
		bits = bits >> 1
		count++
	}
	return result
}
