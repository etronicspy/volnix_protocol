package types

// MaxActivatedLznPerValidator returns the maximum integer amount (min units) one
// validator may hold after a state transition such that 3×amount ≤ totalActivated,
// i.e. at most one third of the aggregate activated pool with integer token amounts.
func MaxActivatedLznPerValidator(totalActivated int64) int64 {
	if totalActivated < 0 {
		return 0
	}
	return totalActivated / 3
}
