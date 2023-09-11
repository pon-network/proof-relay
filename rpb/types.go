package rpbs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"

	rpbsTypes "github.com/bsn-eng/pon-golang-types/rpbs"
	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fp"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

type Signature struct {
	z1_hat *bn254.G1Affine
	c1_hat *big.Int
	s1_hat *big.Int
	c2_hat *big.Int
	s2_hat *big.Int
	m1_hat *bn254.G1Affine
}

type Complex struct {
	re *fp.Element
	im *fp.Element
}

var (
	FIELD_MODULUS  = fp.Modulus()
	CURVE_ORDER    = fr.Modulus()
	_, _, g1Aff, _ = bn254.Generators()
)

func DecodeRPBSSignature(sig *rpbsTypes.EncodedRPBSSignature) (*Signature, error) {
	z1_hat, err := DecodePointFromRPBSFormat(sig.Z1Hat)

	if err != nil {
		return nil, errors.New("Unable to unmarshal z1_hat point from the string")
	}

	c1_hat, ok := new(big.Int).SetString(sig.C1Hat, 16)

	if !ok {
		return nil, errors.New("Unable to get c1_hat from the string")
	}

	s1_hat, ok := new(big.Int).SetString(sig.S1Hat, 16)

	if !ok {
		return nil, errors.New("Unable to get s1_hat from the string")
	}

	c2_hat, ok := new(big.Int).SetString(sig.C2Hat, 16)

	if !ok {
		return nil, errors.New("Unable to get c2_hat from the string")
	}

	s2_hat, ok := new(big.Int).SetString(sig.S2Hat, 16)

	if !ok {
		return nil, errors.New("Unable to get s2_hat from the string")
	}

	m1_hat, err := DecodePointFromRPBSFormat(sig.M1Hat)

	if err != nil {
		return nil, errors.New("Unable to unmarshal m1_hat point from the string")
	}

	return &Signature{z1_hat, c1_hat, s1_hat, c2_hat, s2_hat, m1_hat}, nil
}

// we need encoding a point in this format for compatibility reasons
// since it is used in JS library elliptic.js
// '04' + hex(x) + hex(y)
func EncodePoint(point *bn254.G1Affine) []byte {
	prefix := byte(4)
	encodedPoint := append([]byte{prefix}, point.Marshal()...)
	return encodedPoint
}

// encoded point in RPBS format (that was formed by JS code) is represented
// as a hex string:
// len(hex(X)) + "04" + hex(X) + hex(Y)
func DecodePointFromRPBSFormat(encodedPointInRPBSFormat string) (*bn254.G1Affine, error) {
	point_bytes, err := hex.DecodeString(encodedPointInRPBSFormat)
	if err != nil {
		return nil, errors.New("Unable to decode hex string")
	}

	point := new(bn254.G1Affine)

	// trim first two bytes, i.e. len(hex(X)) + "04"
	if point.Unmarshal(point_bytes[2:]) != nil {
		return nil, errors.New("Unable to unmarshal point from the string")
	}

	return point, nil
}

func ComplexCurveMul(x *Complex, y *Complex, neg *fp.Element) *Complex {
	a1 := x.re
	b1 := x.im

	a2 := y.re
	b2 := y.im

	real := &fp.Element{}
	add := &fp.Element{}

	// real = a1 * a2 + b1 * b2 * neg
	real.Mul(a1, a2).Add(real, add.Mul(b2, neg).Mul(add, b1))

	// imaginary = a1 * b2 + a2 * b1
	imaginary := new(fp.Element)
	imaginary.Mul(a1, b2).Add(imaginary, new(fp.Element).Mul(a2, b1))

	return &Complex{real, imaginary}
}

func ComplexExponent(base *Complex, n *big.Int, neg *fp.Element) *Complex {
	one := fp.One()
	zero := new(fp.Element).SetZero()
	result := Complex{&one, zero}

	base_res := base

	el := &fp.Element{}

	for !zero.Equal(el.SetBigInt(n)) {
		// check whether n is odd
		if n.Bit(0) == 1 {
			result = *ComplexCurveMul(&result, base_res, neg)
		}

		base_res = ComplexCurveMul(base_res, base_res, neg)
		n = n.Rsh(n, 1)
	}

	return &result
}

// x^3 + 3
func CurveRHS(x *fp.Element) *fp.Element {
	res := &fp.Element{}
	_, three := bn254.CurveCoefficients()
	res.Square(x).Mul(res, x).Add(res, &three)

	return res
}

func BytesToPoint(hash [32]byte) (*bn254.G1Affine, error) {
	hashToBigInt := new(big.Int).SetBytes(hash[:])
	reduction := new(big.Int).Mod(hashToBigInt, FIELD_MODULUS)
	x := new(fp.Element).SetBigInt(reduction)

	n := CurveRHS(x)

	// Making sure x^3 + 3 is a square in the field
	one := fp.One()
	zero := new(fp.Element).SetZero()

	for n.Legendre() != 1 {
		x.Add(x, &one)
		n = CurveRHS(x)
	}

	// Cipolla algorithm
	// Making sure a^2 - n (here y^2 = n) is not a square of any number
	a := zero
	neg := new(fp.Element).Neg(n)

	for neg.Legendre() != -1 {
		a.Add(a, &one)
		// a^2 - n
		neg.Square(a).Sub(neg, n)
	}

	/// At this point we should know that a^2 - n is not a square of any number in the field
	base := Complex{a, &one}

	/// Solving for quadratic residue using cipolla's algorithm
	// exp = ( FIELD_MODULUS + 1 ) / 2
	one_bytes := one.Bytes()
	exp := new(big.Int).SetBytes(one_bytes[:])
	exp.Add(FIELD_MODULUS, exp)
	exp.Rsh(exp, 1)

	exponent := ComplexExponent(&base, exp, neg)

	if !exponent.im.Equal(zero) {
		return nil, errors.New("Invalid complex exponent")
	}

	point := bn254.G1Affine{X: *x, Y: *exponent.re}

	if !point.IsOnCurve() {
		return nil, errors.New("Point conversion failed")
	}

	return &point, nil
}

func NegateBigInt(input *big.Int) *big.Int {
	n := new(fr.Element).SetBigInt(input)
	n.Neg(n)
	return new(big.Int).SetBytes(n.Marshal())
}

func VerifySignature(y1 *bn254.G1Affine, info string, sig *Signature) bool {
	infoHash := sha256.Sum256([]byte(info))
	y2, _ := BytesToPoint(infoHash)

	hash := sha256.New()
	hash.Write(EncodePoint(y1))
	hash.Write(EncodePoint(y2))
	hash.Write(EncodePoint(sig.m1_hat))
	hash.Write(EncodePoint(sig.z1_hat))

	p1 := &bn254.G1Affine{}
	p2 := &bn254.G1Affine{}

	// Point::from_scalar(s1_hat) + y1 * -c1_hat
	p1.ScalarMultiplicationBase(sig.s1_hat)
	p2.ScalarMultiplication(y1, NegateBigInt(sig.c1_hat))
	p1.Add(p1, p2)
	hash.Write(EncodePoint(p1))

	/// m1_hat * s1_hat + z1_hat * -c1_hat
	p1.ScalarMultiplication(sig.m1_hat, sig.s1_hat)
	p2.ScalarMultiplication(sig.z1_hat, NegateBigInt(sig.c1_hat))
	p1.Add(p1, p2)
	hash.Write(EncodePoint(p1))

	/// Point::from_scalar(s2_hat) + y2 * -c2_hat
	p1.ScalarMultiplicationBase(sig.s2_hat)
	p2.ScalarMultiplication(y2, NegateBigInt(sig.c2_hat))
	p1.Add(p1, p2)
	hash.Write(EncodePoint(p1))

	hash.Write(EncodePoint(&g1Aff))

	rhs := new(fr.Element).SetBytes(hash.Sum(nil))

	// rhs reduction modulo CURVE_ORDER
	rhs_bytes := rhs.Bytes()
	rhs_big := new(big.Int).SetBytes(rhs_bytes[:])
	z := new(big.Int).Mod(rhs_big, CURVE_ORDER)
	rhs = rhs.SetBigInt(z)

	// computing lhs = c1_hat * c2_hat
	c1_hat := new(fr.Element).SetBigInt(sig.c1_hat)
	c2_hat := new(fr.Element).SetBigInt(sig.c2_hat)
	lhs := new(fr.Element).Mul(c1_hat, c2_hat)

	return lhs.Equal(rhs)
}

func VerifySignatureWithStringInput(rpbsServicePublicKey string, info string, sig *rpbsTypes.EncodedRPBSSignature) (bool, error) {
	y1, err := DecodePointFromRPBSFormat(rpbsServicePublicKey)

	if err != nil {
		return false, errors.New("Unable to unmarshal point from the string")
	}

	signature, err := DecodeRPBSSignature(sig)

	if err != nil {
		return false, errors.New("Unable to decode signature")
	}

	return VerifySignature(y1, info, signature), nil
}
