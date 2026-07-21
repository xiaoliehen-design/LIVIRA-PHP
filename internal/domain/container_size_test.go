package domain

import "testing"

func TestContainerSizeOptionsAndTEU(t *testing.T) {
	valid := []string{"20", "40", "40HC", "45HC", "45"}
	for _, value := range valid {
		if !ValidContainerSize(value) {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	cases := map[string]float64{"20": 1, "40": 2, "40HC": 2, "45": 2.25, "45HC": 2.25}
	for size, want := range cases {
		if got := ContainerTEU(size); got != want {
			t.Fatalf("ContainerTEU(%q) = %v, want %v", size, got, want)
		}
	}
	if got := ContainerSizeLabel("40HC"); got != "40' HC" {
		t.Fatalf("unexpected label: %q", got)
	}
}
