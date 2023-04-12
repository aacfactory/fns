/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sm2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"hash"
	"io"
	"math/big"
	"reflect"
)

var (
	oidSM2               = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	oidPBES2             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}
	oidPBKDF2            = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 12}
	oidKEYMD5            = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 5}
	oidKEYSHA1           = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 7}
	oidKEYSHA256         = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 9}
	oidKEYSHA512         = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 11}
	oidAES128CBC         = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	oidAES256CBC         = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
	oidNamedCurveP256SM2 = asn1.ObjectIdentifier{1, 2, 156, 10197, 1, 301}
)

func GenerateKey(random io.Reader) (*PrivateKey, error) {
	c := P256Sm2()
	if random == nil {
		random = rand.Reader
	}
	params := c.Params()
	b := make([]byte, params.BitSize/8+8)
	_, err := io.ReadFull(random, b)
	if err != nil {
		return nil, err
	}

	k := new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, two)
	k.Mod(k, n)
	k.Add(k, one)
	pri := new(PrivateKey)
	pri.PublicKey.Curve = c
	pri.D = k
	pri.PublicKey.X, pri.PublicKey.Y = c.ScalarBaseMult(k.Bytes())
	return pri, nil
}

func ParsePublicKey(pemBytes []byte) (key *PublicKey, err error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("sm2: pem is invalid")
	}
	if block.Type != "PUBLIC KEY" {
		return nil, errors.New("sm2: block type is not PUBLIC KEY")
	}
	p := block.Bytes

	var ppk pkixPublicKey

	if _, err = asn1.Unmarshal(p, &ppk); err != nil {
		return
	}
	if !reflect.DeepEqual(ppk.Algo.Algorithm, oidSM2) {
		return nil, errors.New("sm2: not sm2 elliptic curve")
	}
	curve := P256Sm2()
	x, y := elliptic.Unmarshal(curve, ppk.BitString.Bytes)
	key = &PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
	return
}

func ParsePrivateKey(pemBytes []byte) (key *PrivateKey, err error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("sm2: pem is invalid")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, errors.New("sm2: block type is not PRIVATE KEY")
	}
	p := block.Bytes

	var priKey pkcs8
	if _, err = asn1.Unmarshal(p, &priKey); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(priKey.Algo.Algorithm, oidSM2) {
		return nil, errors.New("sm2: not sm2 elliptic curve")
	}
	var pk privateKey
	if _, err = asn1.Unmarshal(priKey.PrivateKey, &pk); err != nil {
		return nil, errors.New("sm2: failed to parse SM2 private key: " + err.Error())
	}
	curve := P256Sm2()
	k := new(big.Int).SetBytes(pk.PrivateKey)
	curveOrder := curve.Params().N
	if k.Cmp(curveOrder) >= 0 {
		return nil, errors.New("sm2: invalid elliptic curve private key value")
	}
	key = new(PrivateKey)
	key.Curve = curve
	key.D = k
	pkp := make([]byte, (curveOrder.BitLen()+7)/8)
	for len(pk.PrivateKey) > len(pkp) {
		if pk.PrivateKey[0] != 0 {
			return nil, errors.New("sm2: invalid private key length")
		}
		pk.PrivateKey = pk.PrivateKey[1:]
	}
	copy(pkp[len(pkp)-len(pk.PrivateKey):], pk.PrivateKey)
	key.X, key.Y = curve.ScalarBaseMult(pkp)
	return
}

func ParsePrivateKeyWithPassword(pemBytes []byte, passwd []byte) (key *PrivateKey, err error) {
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, errors.New("sm2: pem is invalid")
	}
	if pemBlock.Type != "ENCRYPTED PRIVATE KEY" {
		return nil, errors.New("sm2: block type is not ENCRYPTED PRIVATE KEY")
	}
	p := pemBlock.Bytes

	var keyInfo encryptedPrivateKeyInfo
	_, err = asn1.Unmarshal(p, &keyInfo)
	if err != nil {
		err = errors.New("sm2: unknown format")
		return
	}
	if !reflect.DeepEqual(keyInfo.EncryptionAlgorithm.IdPBES2, oidPBES2) {
		return nil, errors.New("sm2: only support PBES2")
	}
	encryptionScheme := keyInfo.EncryptionAlgorithm.Pbes2Params.EncryptionScheme
	keyDerivationFunc := keyInfo.EncryptionAlgorithm.Pbes2Params.KeyDerivationFunc
	if !reflect.DeepEqual(keyDerivationFunc.IdPBKDF2, oidPBKDF2) {
		return nil, errors.New("sm2: only support PBKDF2")
	}
	pkdf2Params := keyDerivationFunc.Pkdf2Params
	if !reflect.DeepEqual(encryptionScheme.EncryAlgo, oidAES128CBC) &&
		!reflect.DeepEqual(encryptionScheme.EncryAlgo, oidAES256CBC) {
		return nil, errors.New("sm2: unknown encryption algorithm")
	}
	iv := encryptionScheme.IV
	salt := pkdf2Params.Salt
	iter := pkdf2Params.IterationCount
	encryptedKey := keyInfo.EncryptedData
	var kp []byte
	switch {
	case pkdf2Params.Prf.Algorithm.Equal(oidKEYMD5):
		kp = pbkdf(passwd, salt, iter, 32, md5.New)
		break
	case pkdf2Params.Prf.Algorithm.Equal(oidKEYSHA1):
		kp = pbkdf(passwd, salt, iter, 32, sha1.New)
		break
	case pkdf2Params.Prf.Algorithm.Equal(oidKEYSHA256):
		kp = pbkdf(passwd, salt, iter, 32, sha256.New)
		break
	case pkdf2Params.Prf.Algorithm.Equal(oidKEYSHA512):
		kp = pbkdf(passwd, salt, iter, 32, sha512.New)
		break
	default:
		return nil, errors.New("sm2: unknown hash algorithm")
	}
	block, err := aes.NewCipher(kp)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(encryptedKey, encryptedKey)
	key, err = parsePKCS8PrivateKey(encryptedKey)
	if err != nil {
		err = errors.New("sm2: incorrect password")
		return
	}
	key.password = passwd
	return
}

func parsePKCS8PrivateKey(p []byte) (*PrivateKey, error) {
	var priKey pkcs8

	if _, err := asn1.Unmarshal(p, &priKey); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(priKey.Algo.Algorithm, oidSM2) {
		return nil, errors.New("sm2: not sm2 elliptic curve")
	}
	return parsePrivateKey(priKey.PrivateKey)
}

func parsePrivateKey(p []byte) (*PrivateKey, error) {
	var key privateKey

	if _, err := asn1.Unmarshal(p, &key); err != nil {
		return nil, errors.New("sm2: failed to parse SM2 private key: " + err.Error())
	}
	curve := P256Sm2()
	k := new(big.Int).SetBytes(key.PrivateKey)
	curveOrder := curve.Params().N
	if k.Cmp(curveOrder) >= 0 {
		return nil, errors.New("sm2: invalid elliptic curve private key value")
	}
	pri := new(PrivateKey)
	pri.Curve = curve
	pri.D = k
	pk := make([]byte, (curveOrder.BitLen()+7)/8)
	for len(key.PrivateKey) > len(pk) {
		if key.PrivateKey[0] != 0 {
			return nil, errors.New("sm2: invalid private key length")
		}
		key.PrivateKey = key.PrivateKey[1:]
	}
	copy(pk[len(pk)-len(key.PrivateKey):], key.PrivateKey)
	pri.X, pri.Y = curve.ScalarBaseMult(pk)
	return pri, nil
}

func pbkdf(password, salt []byte, iter, keyLen int, h func() hash.Hash) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var buf [4]byte
	dk := make([]byte, 0, numBlocks*hashLen)
	U := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf[:4])
		dk = prf.Sum(dk)
		T := dk[len(dk)-hashLen:]
		copy(U, T)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(U)
			U = U[:0]
			U = prf.Sum(U)
			for x := range U {
				T[x] ^= U[x]
			}
		}
	}
	return dk[:keyLen]
}

type pkixPublicKey struct {
	Algo      pkix.AlgorithmIdentifier
	BitString asn1.BitString
}

type privateKey struct {
	Version       int
	PrivateKey    []byte
	NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
	PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
}

type pkcs8 struct {
	Version    int
	Algo       pkix.AlgorithmIdentifier
	PrivateKey []byte
}

type pbesKDfs struct {
	IdPBKDF2    asn1.ObjectIdentifier
	Pkdf2Params pkdfParams
}

type pkdfParams struct {
	Salt           []byte
	IterationCount int
	Prf            pkix.AlgorithmIdentifier
}

type pbesEncs struct {
	EncryAlgo asn1.ObjectIdentifier
	IV        []byte
}

type pbesParams struct {
	KeyDerivationFunc pbesKDfs // PBES2-KDFs
	EncryptionScheme  pbesEncs // PBES2-Encs
}

type pbesAlgorithms struct {
	IdPBES2     asn1.ObjectIdentifier
	Pbes2Params pbesParams
}

type encryptedPrivateKeyInfo struct {
	EncryptionAlgorithm pbesAlgorithms
	EncryptedData       []byte
}
