package crypto

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

type NCMRC4 struct {
	keyBox [256]byte // 核心密钥映射表
	keyPos uint8     // 字节流指针位置
}

func NewNCMRC4(key []byte) *NCMRC4 {
	ncm := &NCMRC4{
		keyPos: 0,
	}

	// 1. 初始化标准 S 盒 (0, 1, 2 ... 255)
	var sBox [256]byte
	for i := range 256 {
		sBox[i] = byte(i)
	}

	// 2. 【标准 RC4 初始化】：利用密钥打乱 S 盒
	var j uint8 = 0
	keyLen := len(key)
	for i := range 256 {
		// j = (j + int(sBox[i]) + int(key[i%keyLen])) & 0xFF
		// sBox[i], sBox[j] = sBox[j], sBox[i]
		j = j + sBox[i] + key[i%keyLen]
		sBox[i], sBox[j] = sBox[j], sBox[i]

	}

	// 3. 生成静态密钥字典
	for i := range 256 {
		// idx := (i + 1) & 0xFF
		// s_j := int(sBox[idx])
		// s_jj := int(sBox[(s_j+idx)&0xFF])
		// ncm.keyBox[i] = sBox[(s_jj+s_j)&0xFF]
		idx := uint8(i + 1)
		s_j := sBox[idx]
		s_jj := sBox[s_j+idx]
		ncm.keyBox[i] = sBox[s_jj+s_j]
	}

	return ncm
}

// Decrypt 执行流解密（原地修改）
func (n *NCMRC4) Decrypt(data []byte) {
	for i := range data {
		// 异或解密
		data[i] ^= n.keyBox[n.keyPos]
		// n.keyPos = (n.keyPos + 1) & 0xFF
		n.keyPos = n.keyPos + 1
	}
}

func BenchmarkRC4Reference(b *testing.B) {
	randReader := rand.New(rand.NewSource(1))
	var key [16]byte
	randReader.Read(key[:])
	rc4 := NewNCMRC4(key[:])

	b.SetBytes(100000)
	var buf [100000]byte
	for b.Loop() {
		randReader.Read(buf[:])
		rc4.Decrypt(buf[:])
	}
}

func BenchmarkRC4Streaming(b *testing.B) {
	randReader := rand.New(rand.NewSource(1))
	var key [16]byte
	randReader.Read(key[:])
	rc4 := NewNCMRC4Streaming(randReader, key[:])

	b.SetBytes(100000)
	var buf [100000]byte
	for b.Loop() {
		rc4.Read(buf[:])
	}
}

type fixedSeed struct {
	seed int64
}

func (f *fixedSeed) Int63() int64 {
	return f.seed
}

func (f *fixedSeed) Seed() int64 {
	return f.seed
}

func TestRc4(t *testing.T) {
	seed := &fixedSeed{seed: 1}
	randReader := rand.New(rand.NewSource(seed.seed))
	randReader2 := rand.New(rand.NewSource(seed.seed))
	var key1 [16]byte
	var key2 [16]byte
	randReader.Read(key1[:])
	randReader2.Read(key2[:])
	if !bytes.Equal(key1[:], key2[:]) {
		t.Fatal("keys do not match")
	}

	rc4 := NewNCMRC4(key1[:])
	dec := NewNCMRC4Streaming(randReader, key1[:])

	for range 1000 {
		lenToRead := rand.Intn(512) + 1
		buf := make([]byte, lenToRead)
		buf2 := make([]byte, lenToRead)
		io.ReadFull(randReader2, buf)
		rc4.Decrypt(buf)
		dec.Read(buf2)
		if !bytes.Equal(buf, buf2) {
			t.Fatal("decrypted data does not match original data")
		}
	}

	for range 1000 {
		lenToRead := rand.Intn(200<<10) + 1
		buf := make([]byte, lenToRead)
		buf2 := make([]byte, lenToRead)
		io.ReadFull(randReader2, buf)
		rc4.Decrypt(buf)
		dec.Read(buf2)
		if !bytes.Equal(buf, buf2) {
			t.Fatal("decrypted data does not match original data")
		}
	}
}
