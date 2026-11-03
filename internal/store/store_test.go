package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestInsertAndHistory(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().Unix()
	for i := 0; i < 5; i++ {
		ts := now - int64((4-i)*60)
		if err := st.Insert("web-01", ts, float64(i*10), 1, 2, 3, 4, float64(i)); err != nil {
			t.Fatal(err)
		}
	}

	from := now - 600
	samples, err := st.History("web-01", from, now, 240)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 5 {
		t.Fatalf("samples = %d, want 5", len(samples))
	}
	if samples[4].CPU != 40 {
		t.Errorf("last cpu = %v", samples[4].CPU)
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	old := time.Now().AddDate(0, 0, -40).Unix()
	if err := st.Insert("web-01", old, 1, 1, 1, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := st.Prune(30); err != nil {
		t.Fatal(err)
	}
	samples, err := st.History("web-01", old-60, time.Now().Unix(), 240)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 0 {
		t.Errorf("expected pruned, got %d", len(samples))
	}
}
