package leopard

import (
	"errors"
	"fmt"
)

// Encoder creates parity shards from data shards using Leopard-RS.
type Encoder struct {
	dataShards   int
	parityShards int
	m            int // NextPow2(parityShards)
	n            int // NextPow2(m + dataShards)
}

// New creates a Leopard-RS encoder.
// dataShards + parityShards must be <= 65536.
func New(dataShards, parityShards int) (*Encoder, error) {
	if dataShards < 1 {
		return nil, errors.New("leopard: dataShards must be >= 1")
	}
	if parityShards < 1 {
		return nil, errors.New("leopard: parityShards must be >= 1")
	}
	if dataShards+parityShards > kOrder {
		return nil, fmt.Errorf("leopard: total shards %d exceeds %d", dataShards+parityShards, kOrder)
	}

	initField()

	m := nextPow2(parityShards)
	n := nextPow2(m + dataShards)

	return &Encoder{
		dataShards:   dataShards,
		parityShards: parityShards,
		m:            m,
		n:            n,
	}, nil
}

// Encode generates parity shards from data shards.
// shards[0:dataShards] must contain data, shards[dataShards:] will be filled with parity.
// All shards must be allocated and have equal, even, non-zero length.
func (e *Encoder) Encode(shards [][]byte) error {
	if len(shards) != e.dataShards+e.parityShards {
		return fmt.Errorf("leopard: expected %d shards, got %d", e.dataShards+e.parityShards, len(shards))
	}

	bufBytes := len(shards[0])
	if bufBytes == 0 || bufBytes%2 != 0 {
		return errors.New("leopard: shard size must be even and non-zero")
	}
	for i, s := range shards {
		if len(s) != bufBytes {
			return fmt.Errorf("leopard: shard %d has size %d, expected %d", i, len(s), bufBytes)
		}
	}

	m := e.m

	// Allocate work space: 2*m buffers
	work := make([][]byte, 2*m)
	for i := range work {
		work[i] = make([]byte, bufBytes)
	}

	// Prepare data pointers
	data := make([][]byte, e.dataShards)
	for i := 0; i < e.dataShards; i++ {
		data[i] = shards[i]
	}

	// skewMinus1 simulates C++ "FFTSkew - 1" pointer arithmetic:
	// accessing skewMinus1[i] gives fftSkew[i-1].
	skewMinus1 := make([]uint16, len(fftSkew)+1)
	skewMinus1[0] = 0
	copy(skewMinus1[1:], fftSkew[:])

	// work[0:m] <- IFFT(data[0:m], m, m)
	// C++: skewLUT = FFTSkew + m - 1 → skewMinus1[m:]
	ifftDITEncoder(work[:m], data, min(e.dataShards, m), m, bufBytes, skewMinus1[m:])

	// For remaining sets of m data pieces, XOR in their IFFT
	lastCount := e.dataShards % m
	for i := m; i+m <= e.dataShards; i += m {
		ifftDITEncoderXOR(work[:m], work[m:2*m], data[i:i+m], m, m, bufBytes, skewMinus1[m+i:])
	}

	// Handle final partial set
	if lastCount != 0 && m < e.dataShards {
		offset := (e.dataShards / m) * m
		ifftDITEncoderXOR(work[:m], work[m:2*m], data[offset:offset+lastCount], lastCount, m, bufBytes, skewMinus1[m+offset:])
	}

	// work[0:m] <- FFT(work[0:m], m, 0)
	fftDIT(work[:m], e.parityShards, m, skewMinus1)

	// Copy results to parity shards
	for i := 0; i < e.parityShards; i++ {
		copy(shards[e.dataShards+i], work[i])
	}

	return nil
}

// Decode recovers missing shards. shards[i] == nil means shard i is missing.
// present[i] indicates whether shard i is available.
// On return, all nil shards in [0:dataShards] are recovered.
func (e *Encoder) Decode(shards [][]byte, present []bool) error {
	total := e.dataShards + e.parityShards
	if len(shards) != total || len(present) != total {
		return fmt.Errorf("leopard: expected %d shards/present, got %d/%d", total, len(shards), len(present))
	}

	// Find shard size from any present shard
	bufBytes := 0
	for i, p := range present {
		if p {
			bufBytes = len(shards[i])
			break
		}
	}
	if bufBytes == 0 || bufBytes%2 != 0 {
		return errors.New("leopard: no present shards or invalid shard size")
	}

	// Count missing data and parity
	missingData := 0
	for i := 0; i < e.dataShards; i++ {
		if !present[i] {
			missingData++
		}
	}
	if missingData == 0 {
		return nil // nothing to recover
	}

	missingParity := 0
	for i := e.dataShards; i < total; i++ {
		if !present[i] {
			missingParity++
		}
	}

	// Check if we have enough shards
	totalMissing := missingData + missingParity
	if totalMissing > e.parityShards {
		return fmt.Errorf("leopard: too many shards missing (%d), max recoverable is %d", totalMissing, e.parityShards)
	}

	m := e.m
	n := e.n

	// Separate into original (data) and recovery (parity) for the algorithm
	original := make([][]byte, e.dataShards)
	for i := 0; i < e.dataShards; i++ {
		if present[i] {
			original[i] = shards[i]
		}
	}

	recovery := make([][]byte, e.parityShards)
	for i := 0; i < e.parityShards; i++ {
		if present[e.dataShards+i] {
			recovery[i] = shards[e.dataShards+i]
		}
	}

	// Build error locations
	errorLoc := make([]uint16, kOrder)
	for i := 0; i < e.parityShards; i++ {
		if recovery[i] == nil {
			errorLoc[i] = 1
		}
	}
	for i := e.parityShards; i < m; i++ {
		errorLoc[i] = 1
	}
	for i := 0; i < e.dataShards; i++ {
		if original[i] == nil {
			errorLoc[i+m] = 1
		}
	}

	// Evaluate error locator polynomial via FWHT
	fwht(errorLoc, kOrder, m+e.dataShards)
	for i := 0; i < kOrder; i++ {
		errorLoc[i] = uint16((uint32(errorLoc[i]) * uint32(logWalsh[i])) % kModulus)
	}
	fwht(errorLoc, kOrder, kOrder)

	// Allocate work space
	work := make([][]byte, n)
	for i := range work {
		work[i] = make([]byte, bufBytes)
	}

	// work[0:m] <- recovery data * error_locations
	for i := 0; i < e.parityShards; i++ {
		if recovery[i] != nil {
			mulSlice(work[i], recovery[i], errorLoc[i])
		}
		// else already zeroed
	}

	// work[m:m+dataShards] <- original data * error_locations
	for i := 0; i < e.dataShards; i++ {
		if original[i] != nil {
			mulSlice(work[m+i], original[i], errorLoc[uint16(m+i)])
		}
	}

	// Build skewMinus1 for FFT/IFFT
	skewMinus1 := make([]uint16, len(fftSkew)+1)
	skewMinus1[0] = 0
	copy(skewMinus1[1:], fftSkew[:])

	// IFFT on entire work
	ifftDIT(work, m+e.dataShards, n, skewMinus1)

	// Formal derivative
	for i := 1; i < n; i++ {
		width := ((i ^ (i - 1)) + 1) >> 1
		for j := i - width; j < i; j++ {
			xorSlice(work[j], work[j+width])
		}
	}

	// FFT on work, truncated
	outputCount := m + e.dataShards
	fftDIT(work, outputCount, n, skewMinus1)

	// Reveal erasures
	for i := 0; i < e.dataShards; i++ {
		if original[i] == nil {
			shards[i] = make([]byte, bufBytes)
			mulSlice(shards[i], work[i+m], kModulus-errorLoc[uint16(i+m)])
		}
	}

	return nil
}

// ifftDITEncoder performs IFFT, copying data into work first.
func ifftDITEncoder(work [][]byte, data [][]byte, dataTrunc, m, bufBytes int, skewLUT []uint16) {
	for i := 0; i < dataTrunc; i++ {
		copy(work[i], data[i])
	}
	for i := dataTrunc; i < m; i++ {
		clear(work[i])
	}
	ifftDIT(work, dataTrunc, m, skewLUT)
}

// ifftDITEncoderXOR performs IFFT on data into temp, then XORs into dest.
func ifftDITEncoderXOR(dest, temp, data [][]byte, dataTrunc, m, bufBytes int, skewLUT []uint16) {
	for i := 0; i < dataTrunc; i++ {
		copy(temp[i], data[i])
	}
	for i := dataTrunc; i < m; i++ {
		clear(temp[i])
	}
	ifftDIT(temp, dataTrunc, m, skewLUT)
	for i := 0; i < m; i++ {
		xorSlice(dest[i], temp[i])
	}
}

func nextPow2(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	return v + 1
}
