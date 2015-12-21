package main

type (
	RuneWindow struct {
		arr   [5]rune
		start int // points to an element before the first one
		end   int // point to the last element
	}
)

func add(rw *RuneWindow, r rune) {
	rw.end++
	if rw.end == len(rw.arr) {
		rw.end = 0
	}
	rw.arr[rw.end] = r
	if rw.end == rw.start {
		rw.start++
		if rw.start == len(rw.arr) {
			rw.start = 0
		}
	}
}

func size(rw *RuneWindow) int {
	if rw.end >= rw.start {
		return rw.end - rw.start
	}
	return len(rw.arr) - (rw.start - rw.end)
}

func top(rw *RuneWindow, i int) rune {
	if i >= size(rw) {
		panic("RuneWindow: index out of range")
	}
	index := rw.end - i
	if index < 0 {
		index += len(rw.arr)
	}
	return rw.arr[index]
}
