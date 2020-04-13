package cosh

import (
	"bytes"
	"testing"
)

func TestDeinterlacedOutput(t *testing.T) {
	var exitCode int
	var buf bytes.Buffer

	cfg := DefaultCommandPoolConfig
	cfg.Deinterlace = true
	cfg.Stdout = &buf

	cg := NewCommandPool(&cfg)
	err := cg.Add(2, []string{"echo alpha", "echo beta", "echo delta", "echo gamma"}...)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = cg.Start(0)
	if err != nil {
		t.Fatal(err.Error())
	}
	exitCode, err = cg.Join()
	if err != nil {
		t.Fatal(err.Error())
	}
	if exitCode != 0 {
		t.Fatal("non-zero exit")
	}

	if buf.String() != "alpha\nbeta\ndelta\ngamma\n" {
		t.Fatalf("unexpected output: %v", buf.String())
	}
}
