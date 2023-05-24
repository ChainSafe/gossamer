package erasure_coding

import (
	"github.com/klauspost/reedsolomon"
)

func ObtainChunks(validatorsQty int, data []byte) ([][]byte, error) {
	recoveryThres, err := recoveryThreshold(validatorsQty)
	if err != nil {
		return nil, err
	}
	enc, err := reedsolomon.New(validatorsQty, recoveryThres)
	if err != nil {
		return nil, err
	}
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(shards)
	if err != nil {
		return nil, err
	}

	return shards, nil
}

func recoveryThreshold(validators int) (int, error) {
	needed := (validators - 1) / 3 // TODO add checks for validator < 0
	return needed + 1, nil
}
