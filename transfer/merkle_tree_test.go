package transfer

import (
	"testing"
)

func TestGetEmptyMerkleTree(t *testing.T) {
	tests := []struct {
		name string
		want *MerkleTreeState
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEmptyMerkleTree()
			t.Log(got)
			t.Log(len(got.Layers), got.Layers)
		})
	}
}
