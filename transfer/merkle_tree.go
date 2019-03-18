package transfer

import (
	"crypto/sha256"
	"sort"

	"github.com/oniio/oniChannel/common"
)

type MerkleTreeState struct {
	Layers [][]common.Keccak256
}

func (self *MerkleTreeState) init() {

	self.Layers = append(self.Layers, []common.Keccak256{})
	self.Layers = append(self.Layers, []common.Keccak256{})

	emptyRoot := common.Keccak256{}
	self.Layers[1] = append(self.Layers[1], emptyRoot)
}

func GetEmptyMerkleTree() *MerkleTreeState {
	var merkleTreeState MerkleTreeState
	merkleTreeState.init()
	return &merkleTreeState
}

func HashPair(first common.Keccak256, second common.Keccak256) common.Keccak256 {
	var result common.Keccak256

	empty := common.Keccak256{}

	if common.Keccak256Compare(&first, &empty) == 0 {
		return second
	}

	if common.Keccak256Compare(&second, &empty) == 0 {
		return first
	}

	var data []byte
	if common.Keccak256Compare(&first, &second) > 0 {
		data = append(data, second[:]...)
		data = append(data, first[:]...)
	} else {
		data = append(data, first[:]...)
		data = append(data, second[:]...)
	}

	sum := sha256.Sum256(data)
	result = common.Keccak256(sum)
	return result
}

func computeLayers(elements []common.Keccak256) [][]common.Keccak256 {
	var tree [][]common.Keccak256
	sort.Sort(common.Keccak256Slice(elements))
	leaves := elements

	tree = append(tree, leaves)

	layer := leaves
	for len(layer) > 1 {
		var newLayer []common.Keccak256

		len := len(layer)
		for i := 0; i < len; i += 2 {
			if i+1 < len {
				newLayer = append(newLayer, HashPair(layer[i], layer[i+1]))
			} else {
				newLayer = append(newLayer, HashPair(layer[i], common.Keccak256{}))
			}
		}

		layer = newLayer
		tree = append(tree, layer)
	}

	return tree
}

func computeMerkleProofFor(merkletree [][]common.Keccak256, element common.Keccak256) []common.Keccak256 {
	idx := -1

	leaves := merkletree[0]
	numLeaves := len(leaves)

	for i := 0; i < numLeaves; i++ {
		if common.Keccak256Compare(&leaves[i], &element) == 0 {
			idx = i
			break
		}
	}

	var proof []common.Keccak256
	var numLayers int

	numLayers = len(merkletree)

	for i := 0; i < numLayers; i++ {
		var pair int

		if idx%2 != 0 {
			pair = idx - 1
		} else {
			pair = idx + 1
		}

		if i < len(merkletree[i]) {
			proof = append(proof, merkletree[i][pair])
		}

		idx /= 2
	}

	return proof
}

func validateProof(proof []common.Keccak256, root common.Keccak256, leafElement common.Keccak256) bool {
	hash := leafElement
	len := len(proof)
	for i := 0; i < len; i++ {
		hash = HashPair(hash, proof[i])
	}

	if common.Keccak256Compare(&hash, &root) == 0 {
		return true
	} else {
		return false
	}
}

func MerkleRoot(merkleTree [][]common.Keccak256) common.Keccak256 {
	levelNum := len(merkleTree)
	if levelNum == 0 {
		return common.Keccak256{}
	}

	rootLevel := merkleTree[levelNum-1]
	if len(rootLevel) == 0 {
		return common.Keccak256{}
	}

	return rootLevel[0]

}

func merkleLeveasFromPackedData(packedData []byte) []common.Keccak256 {
	numberOfBytes := len(packedData)

	var leaves []common.Keccak256

	//[TODO] check if 96 = len of leave
	for i := 0; i < numberOfBytes; i = i + 96 {
		sum := sha256.Sum256(packedData[i : i+96])
		leaves = append(leaves, common.Keccak256(sum))
	}

	return leaves

}
