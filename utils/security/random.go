package security

import (
	cryptoRand "crypto/rand"
	"math/rand/v2"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const defaultRandomAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RandomString generates a cryptographically random string of the given length from [A-Za-z0-9]
func RandomString(length int) string {
	return RandomStringWithAlphabet(length, defaultRandomAlphabet)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RandomStringWithAlphabet generates a cryptographically random string using rejection sampling so
// each alphabet character has equal probability
//
// Panics on crypto/rand failure or empty alphabet
func RandomStringWithAlphabet(length int, alphabet string) string {
	if length <= 0 {
		return ""
	}

	n := len(alphabet)
	if n == 0 {
		panic("security: empty alphabet")
	}

	// Bytes in [0, maxByte) map uniformly to indices via % n
	maxByte := 256 - (256 % n)

	out := make([]byte, length)
	var batch [64]byte
	bufPos, bufLen := 0, 0

	readBatch := func() {
		if _, err := cryptoRand.Read(batch[:]); err != nil {
			panic(err)
		}

		bufPos, bufLen = 0, len(batch)
	}

	for i := 0; i < length; {
		if bufPos >= bufLen {
			readBatch()
		}

		rb := batch[bufPos]
		bufPos++

		if int(rb) >= maxByte {
			continue
		}

		out[i] = alphabet[int(rb)%n]
		i++
	}

	return string(out)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// PseudorandomString generates a non-cryptographic random string of the given length from [A-Za-z0-9]
//
// Use RandomString for secrets (tokens, session IDs, etc.)
func PseudorandomString(length int) string {
	return PseudorandomStringWithAlphabet(length, defaultRandomAlphabet)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// PseudorandomStringWithAlphabet is like PseudorandomString with a custom alphabet
//
// Panics if alphabet is empty
func PseudorandomStringWithAlphabet(length int, alphabet string) string {
	if length <= 0 {
		return ""
	}

	n := len(alphabet)
	if n == 0 {
		panic("security: empty alphabet")
	}

	b := make([]byte, length)
	for i := range b {
		b[i] = alphabet[rand.IntN(n)]
	}

	return string(b)
}
