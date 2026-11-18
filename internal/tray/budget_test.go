package tray

import (
	"testing"
)

// TestLayoutBudget is the checked-in performance baseline for the menu:
// hardware-independent and identical on any machine. Raise the budgets only
// for an explained product change.
func TestLayoutBudget(t *testing.T) {
	if raceDetector {
		t.Skip("allocation budgets are not meaningful under the race detector")
	}
	services := layoutServices()
	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		t := &Tray{}
		for i := 0; i < b.N; i++ {
			t.setServices(services)
			t.root()
		}
	})
	const maxAllocs, maxBytes = 245, 15_500
	if got := result.AllocsPerOp(); got > maxAllocs {
		t.Fatalf("layout allocs/op %d exceeds budget %d", got, maxAllocs)
	}
	if got := result.AllocedBytesPerOp(); got > maxBytes {
		t.Fatalf("layout B/op %d exceeds budget %d", got, maxBytes)
	}
	t.Logf("layout allocs/op %d, B/op %d (budget %d / %d)",
		result.AllocsPerOp(), result.AllocedBytesPerOp(), maxAllocs, maxBytes)
}
