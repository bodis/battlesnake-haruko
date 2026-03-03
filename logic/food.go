package logic

// NearestFoodDistance returns the Manhattan distance from start to the
// nearest coordinate in food. Returns -1 if food is empty.
func NearestFoodDistance(start Coord, food []Coord) int {
	if len(food) == 0 {
		return -1
	}
	best := -1
	for _, f := range food {
		d := absInt(start.X-f.X) + absInt(start.Y-f.Y)
		if best < 0 || d < best {
			best = d
		}
	}
	return best
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
