package crypto

import (
	"bytes"
	"testing"
)

func TestReadAndWriteKeypairAsPem(t *testing.T) {
	k, err := GenerateKeyPair()
	if err != nil {
		t.Error(err)
	}
	buf := &bytes.Buffer{}
	if err := WriteKeypairAsPem(buf, k); err != nil {
		t.Error(err)
	}
	kPrime, err := ReadKeypairAsPem(buf)
	if err != nil {
		t.Error(err)
	}

	if k.N.Cmp(kPrime.N) != 0 {
		t.Logf("N: %d != %d\n", k.N, kPrime.N)
		t.Error("original key doesnt match new key")
	}
	if k.E != kPrime.E {
		t.Logf("E: %d != %d\n", k.E, kPrime.E)
		t.Error("original key doesnt match new key")
	}
	if k.D.Cmp(kPrime.D) != 0 {
		t.Logf("D: %d != %d\n", k.D, kPrime.D)
		t.Error("original key doesnt match new key")
	}
}
