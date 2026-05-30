package task

import (
	"encoding/json"
	"testing"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{Pending, "Pending"},
		{Scheduled, "Scheduled"},
		{Running, "Running"},
		{Completed, "Completed"},
		{Failed, "Failed"},
		{State(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestStateMarshalJSON(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{Pending, `"Pending"`},
		{Running, `"Running"`},
		{Completed, `"Completed"`},
	}
	for _, tt := range tests {
		got, err := json.Marshal(tt.state)
		if err != nil {
			t.Fatalf("MarshalJSON(%v): unexpected error: %v", tt.state, err)
		}
		if string(got) != tt.want {
			t.Errorf("MarshalJSON(%v) = %s, want %s", tt.state, got, tt.want)
		}
	}
}

func TestStateUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  State
	}{
		{`"Pending"`, Pending},
		{`"Scheduled"`, Scheduled},
		{`"Running"`, Running},
		{`"Completed"`, Completed},
		{`"Failed"`, Failed},
	}
	for _, tt := range tests {
		var s State
		if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
			t.Fatalf("UnmarshalJSON(%s): unexpected error: %v", tt.input, err)
		}
		if s != tt.want {
			t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, s, tt.want)
		}
	}
}

func TestStateUnmarshalJSONUnknown(t *testing.T) {
	var s State
	if err := json.Unmarshal([]byte(`"Bogus"`), &s); err == nil {
		t.Error("expected error for unknown state, got nil")
	}
}

func TestStateMarshalRoundTrip(t *testing.T) {
	for _, original := range []State{Pending, Scheduled, Running, Completed, Failed} {
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", original, err)
		}
		var decoded State
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal(%v): %v", original, err)
		}
		if decoded != original {
			t.Errorf("round-trip %v → %v", original, decoded)
		}
	}
}

func TestValidStateTransition(t *testing.T) {
	valid := []struct{ src, dst State }{
		{Pending, Scheduled},
		{Scheduled, Scheduled},
		{Scheduled, Running},
		{Scheduled, Failed},
		{Running, Running},
		{Running, Completed},
		{Running, Failed},
	}
	for _, tt := range valid {
		if !ValidStateTransition(tt.src, tt.dst) {
			t.Errorf("expected valid transition %v → %v", tt.src, tt.dst)
		}
	}

	invalid := []struct{ src, dst State }{
		{Pending, Running},
		{Pending, Completed},
		{Pending, Failed},
		{Completed, Running},
		{Completed, Scheduled},
		{Failed, Running},
		{Failed, Completed},
	}
	for _, tt := range invalid {
		if ValidStateTransition(tt.src, tt.dst) {
			t.Errorf("expected invalid transition %v → %v", tt.src, tt.dst)
		}
	}
}
