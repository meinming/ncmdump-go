package crypto

import (
	"crypto/subtle"
	"io"
)

type NCMRC4Streaming struct {
	keyBox [256]uint8 // 核心密钥映射表
	keyPos uint8      // 字节流指针位置
	reader io.Reader
}

func NewNCMRC4Streaming(reader io.Reader, key []byte) *NCMRC4Streaming {
	ncm := &NCMRC4Streaming{
		keyPos: 0,
		reader: reader,
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

func (n *NCMRC4Streaming) Close() error {
	if closer, ok := n.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (n *NCMRC4Streaming) Read(p []byte) (int, error) {
	nBytes, err := n.reader.Read(p)
	if err != nil {
		return 0, err
	}

	p = p[:nBytes]

	for len(p) > 0 {
		l := subtle.XORBytes(p, p, n.keyBox[n.keyPos:])
		n.keyPos += uint8(l)
		p = p[l:]
	}

	return nBytes, nil
}
