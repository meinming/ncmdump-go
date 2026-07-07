package crypto

type NCMRC4 struct {
	key    []byte
	keyBox [256]byte // 核心密钥映射表
	keyPos uint8     // 字节流指针位置
}

func NewNCMRC4(key []byte) *NCMRC4 {
	ncm := &NCMRC4{
		key:    key,
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
