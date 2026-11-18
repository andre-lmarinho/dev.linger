package monitor

import (
	"os"
	"testing"
)

// TestDiscoveryBudget is the checked-in performance baseline. It counts only
// hardware-independent work, so it produces the same result on any machine:
// a US$ 200 laptop and a US$ 40,000 workstation enforce identical budgets.
// The fixture is built outside the measured closure so its construction cost
// never leaks into the per-operation numbers. Raise the budgets only for an
// explained product change.
func TestDiscoveryBudget(t *testing.T) {
	if raceDetector {
		t.Skip("allocation budgets are not meaningful under the race detector")
	}
	root := controlledProc(t)
	containers := controlledContainers()
	var benchErr error
	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			listeners, err := readListeners(root)
			if err != nil {
				benchErr = err
				return
			}
			collect(listeners, processOwners(root, os.Getuid(), listeners), containers)
		}
	})
	if benchErr != nil {
		t.Fatal(benchErr)
	}
	const maxAllocs, maxBytes = 15_200, 1_400_000
	if got := result.AllocsPerOp(); got > maxAllocs {
		t.Fatalf("discovery allocs/op %d exceeds budget %d", got, maxAllocs)
	}
	if got := result.AllocedBytesPerOp(); got > maxBytes {
		t.Fatalf("discovery B/op %d exceeds budget %d", got, maxBytes)
	}
	t.Logf("discovery allocs/op %d, B/op %d (budget %d / %d)",
		result.AllocsPerOp(), result.AllocedBytesPerOp(), maxAllocs, maxBytes)
}
