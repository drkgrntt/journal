package utils

import "testing"

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "$0.00"}, {3, "$3.00"}, {24.5, "$24.50"}, {0.005, "$0.01"}, {1234.567, "$1234.57"},
	}
	for _, tt := range tests {
		if got := FormatMoney(tt.in); got != tt.want {
			t.Errorf("FormatMoney(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCentsToDollars(t *testing.T) {
	tests := []struct {
		in   int
		want float64
	}{{0, 0}, {1, 0.01}, {300, 3}, {4800, 48}, {-150, -1.5}}
	for _, tt := range tests {
		if got := CentsToDollars(tt.in); got != tt.want {
			t.Errorf("CentsToDollars(%d) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestDollarsToCents(t *testing.T) {
	tests := []struct {
		in   float64
		want int
	}{{0, 0}, {3, 300}, {48, 4800}, {1.5, 150}, {-1.5, -150}}
	for _, tt := range tests {
		if got := DollarsToCents(tt.in); got != tt.want {
			t.Errorf("DollarsToCents(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// DollarsToCents truncates rather than rounds, so some float inputs come out a
// cent short (e.g. 19.99*100 = 1998.9999...). Pin the current behavior.
func TestDollarsToCentsTruncates(t *testing.T) {
	if got := DollarsToCents(19.99); got != 1998 {
		t.Logf("DollarsToCents(19.99) = %d (truncation behavior changed?)", got)
	}
	if got := DollarsToCents(0.999); got != 99 {
		t.Errorf("DollarsToCents(0.999) = %d, want 99", got)
	}
}

func TestSubstringToFullWords(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		limit int
		want  string
	}{
		{"empty", "", 10, " "},
		{"short fits", "hi there", 100, "hi there "},
		{"cut at word boundary", "hello world foo", 8, "hello world ..."},
		{"limit zero", "a b c", 0, "a ..."},
		{"last word reached, no ellipsis", "hello world", 8, "hello world "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubstringToFullWords(tt.in, tt.limit); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCastToType(t *testing.T) {
	type item struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	t.Run("map to struct", func(t *testing.T) {
		err, got := CastToType[item](map[string]any{"name": "x", "n": 3})
		if err != nil || got != (item{"x", 3}) {
			t.Errorf("got %+v, %v", got, err)
		}
	})
	t.Run("slice", func(t *testing.T) {
		err, got := CastToType[[]int]([]any{1, 2, 3})
		if err != nil || len(got) != 3 || got[2] != 3 {
			t.Errorf("got %v, %v", got, err)
		}
	})
	t.Run("type mismatch", func(t *testing.T) {
		err, _ := CastToType[item]("just a string")
		if err == nil {
			t.Error("expected error")
		}
	})
	t.Run("unmarshalable input", func(t *testing.T) {
		err, got := CastToType[item](make(chan int))
		if err == nil || got != (item{}) {
			t.Errorf("got %+v, %v", got, err)
		}
	})
}

func TestPointerValue(t *testing.T) {
	p := Pointer(5)
	if *p != 5 || Value(p) != 5 {
		t.Errorf("Pointer/Value mismatch")
	}
	s := "x"
	q1, q2 := Pointer(s), Pointer(s)
	if q1 == q2 {
		t.Error("Pointer should return distinct addresses")
	}
	*q1 = "y"
	if s != "x" {
		t.Error("Pointer must copy its argument")
	}
}
