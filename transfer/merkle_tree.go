package transfer

import (
	"crypto/sha256"
	"github.com/oniio/oniChannel/typing"
	"sort"
)

func HashPair(first typing.Keccak256, second typing.Keccak256) typing.Keccak256 {
	var result typing.Keccak256

	empty := typing.Keccak256{}

	if typing.Keccak256Compare(&first, &empty) == 0 {
		return second
	}

	if typing.Keccak256Compare(&second, &empty) == 0 {
		return first
	}

	if typing.Keccak256Compare(&first, &first) > 0 {
		data := []byte{}

		data = append(data, first[:]...)
		data = append(data, second[:]...)

		sum := sha256.Sum256(data)
		result = typing.Keccak256(sum)
	}

	return result
}

func computeLayers(elements []typing.Keccak256) [][]typing.Keccak256 {

	var tree [][]typing.Keccak256

	sort.Sort(typing.Keccak256Slice(elements))
	leaves := elements

	tree = append(tree, leaves)

	layer := leaves
	for len(layer) > 1 {
		newLayer := []typing.Keccak256{}

		len := len(layer)
		for i := 0; i < len; i += 2 {
			if i+1 < len {
				newLayer = append(newLayer, HashPair(layer[i], layer[i+1]))
			} else {
				newLayer = append(newLayer, HashPair(layer[i], typing.Keccak256{}))
			}
		}

		layer = newLayer
		tree = append(tree, layer)
	}

	return tree
}

func computeMerkleProofFor(merkletree [][]typing.Keccak256, element typing.Keccak256) []typing.Keccak256 {
	idx := -1

	leaves := merkletree[0]
	numLeaves := len(leaves)

	for i := 0; i < numLeaves; i++ {
		if typing.Keccak256Compare(&leaves[i], &element) == 0 {
			idx = i
			break
		}
	}

	var proof []typing.Keccak256
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

func validateProof(proof []typing.Keccak256, root typing.Keccak256, leafElement typing.Keccak256) bool {
	hash := leafElement
	len := len(proof)
	for i := 0; i < len; i++ {
		hash = HashPair(hash, proof[i])
	}

	if typing.Keccak256Compare(&hash, &root) == 0 {
		return true
	} else {
		return false
	}
}

func merkelRoot(merkletree [][]typing.Keccak256) typing.Keccak256 {
	levelNum := len(merkletree)
	if levelNum == 0 {
		return typing.Keccak256{}
	}

	rootLevel := merkletree[levelNum-1]
	if len(rootLevel) == 0 {
		return typing.Keccak256{}
	}

	return rootLevel[0]

}

func merkelLeveasFromPackedData(packedData []byte) []typing.Keccak256 {
	numberOfBytes := len(packedData)

	leaves := []typing.Keccak256{}

	//[TODO] check if 96 = len of leave
	for i := 0; i < numberOfBytes; i = i + 96 {
		sum := sha256.Sum256(packedData[i : i+96])
		leaves = append(leaves, typing.Keccak256(sum))
	}

	return leaves

}
