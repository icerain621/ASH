package passk

import (
	"fmt"
	"testing"
)

func TestPassKAllPass(t *testing.T) {
	res, err := Run(Config{K: 3, Check: func(int) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if !res.AllPass || res.Passes != 3 || res.Fails != 0 {
		t.Fatalf("%+v", res)
	}
}

func TestPassKClustersFailures(t *testing.T) {
	res, err := Run(Config{K: 4, Check: func(i int) error {
		if i%2 == 0 {
			return fmt.Errorf("format")
		}
		return fmt.Errorf("tool")
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res.AllPass || res.Fails != 4 || len(res.FailClusters) != 2 {
		t.Fatalf("%+v", res)
	}
}

func TestPassKRejectsZero(t *testing.T) {
	if _, err := Run(Config{K: 0, Check: func(int) error { return nil }}); err == nil {
		t.Fatal("expected error")
	}
}
