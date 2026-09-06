package rag

import "testing"

func TestApproximateTokenCounter_Count(t *testing.T) {
	counter := ApproximateTokenCounter{}

	tests := []struct {
		name string
		text string
		want int
	}{
		{
			name: "empty text",
			text: "",
			want: 0,
		},
		{
			name: "exact multiple",
			text: "12345678",
			want: 2,
		},
		{
			name: "rounds up",
			text: "12345",
			want: 2,
		},
		{
			name: "single character",
			text: "a",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := counter.Count(tt.text)

				if got != tt.want {
					t.Fatalf("expected %d tokens, got %d", tt.want, got)
				}
			},
		)
	}
}
