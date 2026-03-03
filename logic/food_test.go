package logic

import "testing"

func TestNearestFoodDistanceBasic(t *testing.T) {
	got := NearestFoodDistance(Coord{5, 5}, []Coord{{5, 8}})
	if got != 3 {
		t.Errorf("NearestFoodDistance = %d, want 3", got)
	}
}

func TestNearestFoodDistanceMultiple(t *testing.T) {
	got := NearestFoodDistance(Coord{5, 5}, []Coord{{5, 8}, {5, 6}, {0, 0}})
	if got != 1 {
		t.Errorf("NearestFoodDistance = %d, want 1", got)
	}
}

func TestNearestFoodDistanceNoFood(t *testing.T) {
	got := NearestFoodDistance(Coord{5, 5}, nil)
	if got != -1 {
		t.Errorf("NearestFoodDistance = %d, want -1", got)
	}
}

func TestNearestFoodDistanceSamePosition(t *testing.T) {
	got := NearestFoodDistance(Coord{3, 3}, []Coord{{3, 3}})
	if got != 0 {
		t.Errorf("NearestFoodDistance = %d, want 0", got)
	}
}

func TestNearestFoodDistanceMaxDistance(t *testing.T) {
	got := NearestFoodDistance(Coord{0, 0}, []Coord{{10, 10}})
	if got != 20 {
		t.Errorf("NearestFoodDistance = %d, want 20", got)
	}
}
