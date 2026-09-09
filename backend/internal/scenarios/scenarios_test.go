package scenarios

import (
	"testing"
)

func TestScenariosListAndBuild(t *testing.T) {
	list := List()
	if len(list) < 5 {
		t.Fatalf("expected at least 5 scenarios, got %d", len(list))
	}

	for _, sc := range list {
		sessID, events, err := BuildEvents(sc.ID)
		if err != nil {
			t.Errorf("failed to build scenario %s: %v", sc.ID, err)
		}
		if sessID == "" || len(events) == 0 {
			t.Errorf("scenario %s built empty events", sc.ID)
		}
	}
}
