package leopard

import "unsafe"

// ifftButterfly performs the IFFT butterfly:
//   y[] ^= x[]
//   x[] ^= y[] * exp(logM)
func ifftButterfly(x, y []byte, logM uint16) {
	xorSlice(y, x)
	if logM != kModulus {
		mulAddSlice(x, y, logM)
	}
}

// fftButterfly performs the FFT butterfly:
//   x[] ^= y[] * exp(logM)
//   y[] ^= x[]
func fftButterfly(x, y []byte, logM uint16) {
	if logM != kModulus {
		mulAddSlice(x, y, logM)
	}
	xorSlice(y, x)
}

// xorSlice: dst[i] ^= src[i]
func xorSlice(dst, src []byte) {
	n := len(dst) / 8
	dstW := unsafe.Slice((*uint64)(unsafe.Pointer(&dst[0])), n)
	srcW := unsafe.Slice((*uint64)(unsafe.Pointer(&src[0])), n)
	for i := range dstW {
		dstW[i] ^= srcW[i]
	}
	for i := n * 8; i < len(dst); i++ {
		dst[i] ^= src[i]
	}
}

// mulAddSlice: dst[i] ^= src[i] * exp(logM) for each 16-bit element
func mulAddSlice(dst, src []byte, logM uint16) {
	n := len(dst) / 2
	dstW := unsafe.Slice((*uint16)(unsafe.Pointer(&dst[0])), n)
	srcW := unsafe.Slice((*uint16)(unsafe.Pointer(&src[0])), n)
	for i := 0; i < n; i++ {
		if srcW[i] != 0 {
			dstW[i] ^= expLUT[addMod(logLUT[srcW[i]], logM)]
		}
	}
}

// mulSlice: dst[i] = src[i] * exp(logM) for each 16-bit element
func mulSlice(dst, src []byte, logM uint16) {
	n := len(dst) / 2
	dstW := unsafe.Slice((*uint16)(unsafe.Pointer(&dst[0])), n)
	srcW := unsafe.Slice((*uint16)(unsafe.Pointer(&src[0])), n)
	for i := 0; i < n; i++ {
		if srcW[i] == 0 {
			dstW[i] = 0
		} else {
			dstW[i] = expLUT[addMod(logLUT[srcW[i]], logM)]
		}
	}
}

// ifftDIT performs the inverse FFT (decimation in time) on work buffers.
func ifftDIT(work [][]byte, mTruncated, m int, skewLUT []uint16) {
	bytes := len(work[0])

	dist := 1
	for dist4 := 4; dist4 <= m; dist4 <<= 2 {
		for r := 0; r < mTruncated; r += dist4 {
			iEnd := r + dist
			logM01 := skewLUT[iEnd]
			logM02 := skewLUT[iEnd+dist]
			logM23 := skewLUT[iEnd+dist*2]

			for i := r; i < iEnd; i++ {
				// 4-way IFFT butterfly
				ifftDIT4(work, i, dist, logM01, logM23, logM02, bytes)
			}
		}
		dist = dist4
	}

	if dist < m {
		logM := skewLUT[dist]
		if logM == kModulus {
			for i := 0; i < dist; i++ {
				xorSlice(work[i+dist], work[i])
			}
		} else {
			for i := 0; i < dist; i++ {
				ifftButterfly(work[i], work[i+dist], logM)
			}
		}
	}
}

func ifftDIT4(work [][]byte, i, dist int, logM01, logM23, logM02 uint16, bytes int) {
	// First layer
	if logM01 == kModulus {
		xorSlice(work[i+dist], work[i])
	} else {
		ifftButterfly(work[i], work[i+dist], logM01)
	}

	if logM23 == kModulus {
		xorSlice(work[i+dist*3], work[i+dist*2])
	} else {
		ifftButterfly(work[i+dist*2], work[i+dist*3], logM23)
	}

	// Second layer
	if logM02 == kModulus {
		xorSlice(work[i+dist*2], work[i])
		xorSlice(work[i+dist*3], work[i+dist])
	} else {
		ifftButterfly(work[i], work[i+dist*2], logM02)
		ifftButterfly(work[i+dist], work[i+dist*3], logM02)
	}
}

// fftDIT performs the forward FFT (decimation in time) on work buffers.
func fftDIT(work [][]byte, mTruncated, m int, skewLUT []uint16) {
	dist4 := m
	dist := m >> 2
	for dist > 0 {
		for r := 0; r < mTruncated; r += dist4 {
			iEnd := r + dist
			logM01 := skewLUT[iEnd]
			logM02 := skewLUT[iEnd+dist]
			logM23 := skewLUT[iEnd+dist*2]

			for i := r; i < iEnd; i++ {
				fftDIT4(work, i, dist, logM01, logM23, logM02)
			}
		}
		dist4 = dist
		dist >>= 2
	}

	if dist4 == 2 {
		for r := 0; r < mTruncated; r += 2 {
			logM := skewLUT[r+1]
			if logM == kModulus {
				xorSlice(work[r+1], work[r])
			} else {
				fftButterfly(work[r], work[r+1], logM)
			}
		}
	}
}

func fftDIT4(work [][]byte, i, dist int, logM01, logM23, logM02 uint16) {
	// First layer (large distance)
	if logM02 == kModulus {
		xorSlice(work[i+dist*2], work[i])
		xorSlice(work[i+dist*3], work[i+dist])
	} else {
		fftButterfly(work[i], work[i+dist*2], logM02)
		fftButterfly(work[i+dist], work[i+dist*3], logM02)
	}

	// Second layer (small distance)
	if logM01 == kModulus {
		xorSlice(work[i+dist], work[i])
	} else {
		fftButterfly(work[i], work[i+dist], logM01)
	}

	if logM23 == kModulus {
		xorSlice(work[i+dist*3], work[i+dist*2])
	} else {
		fftButterfly(work[i+dist*2], work[i+dist*3], logM23)
	}
}
