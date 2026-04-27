// Package leopard implements the Leopard-RS erasure coding algorithm using
// additive FFT over GF(2^16). This provides O(n log n) encoding and decoding.
//
// Reference: https://github.com/catid/leopard (Christopher A. Taylor)
package leopard

// Leopard-RS uses its own GF(2^16) with LFSR polynomial 0x1002D and a
// specific Cantor basis. This is different from the project's gf65536
// package which uses 0x1002B. The Cantor basis values are tightly coupled
// to the polynomial choice, so we use self-contained field arithmetic here.

const (
	kBits       = 16
	kOrder      = 65536
	kModulus    = 65535 // kOrder - 1, also used as sentinel for "no multiply"
	kPolynomial = 0x1002D
)

// Cantor basis for GF(2^16) with polynomial 0x1002D.
// This specific basis enables the additive FFT structure.
var cantorBasis = [kBits]uint16{
	0x0001, 0xACCA, 0x3C0E, 0x163E,
	0xC582, 0xED2E, 0x914C, 0x4012,
	0x6C98, 0x10D8, 0x6A72, 0xB900,
	0xFDB8, 0xFB34, 0xFF38, 0x991E,
}

var (
	logLUT [kOrder]uint16
	expLUT [kOrder + 1]uint16 // +1 so expLUT[kModulus] is valid

	// FFT skew factors in log domain
	fftSkew [kModulus]uint16

	// FWHT of log table, used for error locator polynomial
	logWalsh [kOrder]uint16

	initialized bool
)

func initField() {
	if initialized {
		return
	}

	// Generate exp/log tables via LFSR
	var state uint32 = 1
	for i := uint32(0); i < kModulus; i++ {
		expLUT[state] = uint16(i)
		state <<= 1
		if state >= kOrder {
			state ^= kPolynomial
		}
	}
	expLUT[0] = kModulus

	// Convert to Cantor basis representation
	logLUT[0] = 0
	for i := 0; i < kBits; i++ {
		basis := cantorBasis[i]
		width := uint32(1) << i
		for j := uint32(0); j < width; j++ {
			logLUT[j+width] = logLUT[j] ^ basis
		}
	}

	// Apply exp to convert from Cantor basis index to log
	for i := 0; i < kOrder; i++ {
		logLUT[i] = expLUT[logLUT[i]]
	}
	// Build reverse mapping
	for i := 0; i < kOrder; i++ {
		expLUT[logLUT[i]] = uint16(i)
	}
	expLUT[kModulus] = expLUT[0]

	initFFTSkew()
	initLogWalsh()

	initialized = true
}

// addMod returns (a + b) mod kModulus, allowing kModulus as output.
func addMod(a, b uint16) uint16 {
	sum := uint32(a) + uint32(b)
	return uint16(sum + (sum >> kBits))
}

// subMod returns (a - b) mod kModulus.
func subMod(a, b uint16) uint16 {
	dif := uint32(a) - uint32(b)
	return uint16(dif + (dif >> kBits))
}

// mulLog returns a * exp(log_b), where log_b is already a logarithm.
func mulLog(a, log_b uint16) uint16 {
	if a == 0 {
		return 0
	}
	return expLUT[addMod(logLUT[a], log_b)]
}

func initFFTSkew() {
	temp := make([]uint16, kBits-1)
	for i := 1; i < kBits; i++ {
		temp[i-1] = uint16(1 << i)
	}

	for m := 0; m < kBits-1; m++ {
		step := uint32(1) << (m + 1)
		fftSkew[(1<<m)-1] = 0

		for i := m; i < kBits-1; i++ {
			s := uint32(1) << (i + 1)
			for j := uint32((1 << m) - 1); j < s; j += step {
				fftSkew[j+s] = fftSkew[j] ^ temp[i]
			}
		}

		// temp[m] = kModulus - log(temp[m] * log(temp[m] ^ 1))
		temp[m] = kModulus - logLUT[mulLog(temp[m], logLUT[temp[m]^1])]

		for i := m + 1; i < kBits-1; i++ {
			sum := addMod(logLUT[temp[i]^1], temp[m])
			temp[i] = mulLog(temp[i], sum)
		}
	}

	// Convert to log domain
	for i := 0; i < kModulus; i++ {
		fftSkew[i] = logLUT[fftSkew[i]]
	}
}

func initLogWalsh() {
	for i := 0; i < kOrder; i++ {
		logWalsh[i] = logLUT[i]
	}
	logWalsh[0] = 0
	fwht(logWalsh[:], kOrder, kOrder)
}

// fwht performs the Fast Walsh-Hadamard Transform on GF element logs.
func fwht(data []uint16, m, mTruncated int) {
	dist := 1
	for dist4 := 4; dist4 <= m; dist4 <<= 2 {
		for r := 0; r < mTruncated; r += dist4 {
			for i := r; i < r+dist; i++ {
				fwht4(data, i, dist)
			}
		}
		dist = dist4
	}
	if dist < m {
		for i := 0; i < dist; i++ {
			fwht2(&data[i], &data[i+dist])
		}
	}
}

func fwht2(a, b *uint16) {
	sum := addMod(*a, *b)
	dif := subMod(*a, *b)
	*a = sum
	*b = dif
}

func fwht4(data []uint16, i, s int) {
	s2 := s << 1
	t0 := data[i]
	t1 := data[i+s]
	t2 := data[i+s2]
	t3 := data[i+s2+s]

	// Two layers of FWHT butterflies
	sum01 := addMod(t0, t1)
	dif01 := subMod(t0, t1)
	sum23 := addMod(t2, t3)
	dif23 := subMod(t2, t3)

	data[i] = addMod(sum01, sum23)
	data[i+s] = addMod(dif01, dif23)
	data[i+s2] = subMod(sum01, sum23)
	data[i+s2+s] = subMod(dif01, dif23)
}
