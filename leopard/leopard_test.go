package leopard

import (
	"crypto/rand"
	"testing"
)

func TestLeopardBasic(t *testing.T) {
	enc, err := New(4, 2)
	if err != nil {
		t.Fatal(err)
	}

	shardSize := 1024
	shards := make([][]byte, 6)
	for i := 0; i < 4; i++ {
		shards[i] = make([]byte, shardSize)
		rand.Read(shards[i])
	}
	shards[4] = make([]byte, shardSize)
	shards[5] = make([]byte, shardSize)

	if err := enc.Encode(shards); err != nil {
		t.Fatal("Encode:", err)
	}

	// Verify parity is not all zeros
	allZero := true
	for _, b := range shards[4] {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("parity shard 0 is all zeros")
	}

	// Save originals
	orig := make([][]byte, 4)
	for i := 0; i < 4; i++ {
		orig[i] = make([]byte, shardSize)
		copy(orig[i], shards[i])
	}

	// Erase 2 data shards
	shards[1] = nil
	shards[3] = nil
	present := []bool{true, false, true, false, true, true}

	if err := enc.Decode(shards, present); err != nil {
		t.Fatal("Decode:", err)
	}

	// Verify recovered
	for _, idx := range []int{1, 3} {
		if shards[idx] == nil {
			t.Fatalf("shard %d not recovered", idx)
		}
		for j := range orig[idx] {
			if shards[idx][j] != orig[idx][j] {
				t.Fatalf("shard %d mismatch at byte %d: got %02x want %02x", idx, j, shards[idx][j], orig[idx][j])
			}
		}
	}
}

func TestLeopardNoErasure(t *testing.T) {
	enc, err := New(4, 2)
	if err != nil {
		t.Fatal(err)
	}

	shards := make([][]byte, 6)
	for i := range shards {
		shards[i] = make([]byte, 64)
		rand.Read(shards[i])
	}
	present := []bool{true, true, true, true, true, true}
	if err := enc.Decode(shards, present); err != nil {
		t.Fatal(err)
	}
}

func TestLeopardEraseParity(t *testing.T) {
	enc, err := New(4, 4)
	if err != nil {
		t.Fatal(err)
	}

	shardSize := 512
	shards := make([][]byte, 8)
	for i := 0; i < 4; i++ {
		shards[i] = make([]byte, shardSize)
		rand.Read(shards[i])
	}
	for i := 4; i < 8; i++ {
		shards[i] = make([]byte, shardSize)
	}

	if err := enc.Encode(shards); err != nil {
		t.Fatal(err)
	}

	orig := make([][]byte, 4)
	for i := 0; i < 4; i++ {
		orig[i] = make([]byte, shardSize)
		copy(orig[i], shards[i])
	}

	// Erase 2 data + 2 parity
	shards[0] = nil
	shards[2] = nil
	shards[5] = nil
	shards[7] = nil
	present := []bool{false, true, false, true, true, false, true, false}

	if err := enc.Decode(shards, present); err != nil {
		t.Fatal(err)
	}

	for _, idx := range []int{0, 2} {
		if shards[idx] == nil {
			t.Fatalf("shard %d not recovered", idx)
		}
		for j := range orig[idx] {
			if shards[idx][j] != orig[idx][j] {
				t.Fatalf("shard %d mismatch at byte %d", idx, j)
			}
		}
	}
}

func TestLeopardMedium(t *testing.T) {
	enc, err := New(100, 50)
	if err != nil {
		t.Fatal(err)
	}

	shardSize := 4096
	total := 150
	shards := make([][]byte, total)
	for i := 0; i < 100; i++ {
		shards[i] = make([]byte, shardSize)
		rand.Read(shards[i])
	}
	for i := 100; i < total; i++ {
		shards[i] = make([]byte, shardSize)
	}

	if err := enc.Encode(shards); err != nil {
		t.Fatal(err)
	}

	// Save originals
	orig := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		orig[i] = make([]byte, shardSize)
		copy(orig[i], shards[i])
	}

	// Erase 30 data shards (deterministic pattern)
	present := make([]bool, total)
	for i := range present {
		present[i] = true
	}
	erased := 0
	for i := 0; i < 100 && erased < 30; i += 3 {
		shards[i] = nil
		present[i] = false
		erased++
	}

	if err := enc.Decode(shards, present); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		if !present[i] {
			// was erased, verify recovery
			continue // present was modified... check via orig
		}
	}
	// Verify all erased shards
	for i := 0; i < 100; i += 3 {
		if i/3 >= 30 {
			break
		}
		if shards[i] == nil {
			t.Fatalf("shard %d not recovered", i)
		}
		for j := range orig[i] {
			if shards[i][j] != orig[i][j] {
				t.Fatalf("shard %d mismatch at byte %d", i, j)
			}
		}
	}
}

func TestLeopardValidation(t *testing.T) {
	_, err := New(0, 5)
	if err == nil {
		t.Fatal("expected error for 0 data shards")
	}
	_, err = New(5, 0)
	if err == nil {
		t.Fatal("expected error for 0 parity shards")
	}
	_, err = New(40000, 30000)
	if err == nil {
		t.Fatal("expected error for exceeding 65536")
	}
}
