package agent

import "testing"

func TestGetAgent(t *testing.T) {
	data, err := GetAgent()
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Agent binary is empty")
	}
}
