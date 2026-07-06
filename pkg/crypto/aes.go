package crypto

import (
	"bytes"
	"crypto/aes"
	"fmt"
)

var aesKeyRc4Key = []byte{
	0x68, 0x7A, 0x48, 0x52, 0x41, 0x6D, 0x73, 0x6F,
	0x35, 0x6B, 0x49, 0x6E, 0x62, 0x61, 0x78, 0x57,
} // "687A4852416D736F356B496E62617857"
const rc4KeyXorByte byte = 0x64

var prefixToRemove = []byte("neteasecloudmusic")

type Decoder struct {
	Rc4KeyEnc       []byte
	Rc4Key          []byte
	MetadataEncSize uint32
	MetadataEnc     []byte
	Metadata        *NcmMetadata
}

// DecryptRC4Key 执行解密核心逻辑
func (d *Decoder) DecryptRC4Key() error {
	// 异或混淆取反
	xorData := make([]byte, len(d.Rc4KeyEnc))
	for i, b := range d.Rc4KeyEnc {
		xorData[i] = b ^ rc4KeyXorByte
	}

	// AES-ECB 解密
	decryptedData, err := aesECBDecrypt(xorData, aesKeyRc4Key)
	if err != nil {
		return fmt.Errorf("AES 解密失败: %w", err)
	}

	// 去除 PKCS7 填充
	unpaddedData, err := pkcs7Unpad(decryptedData)
	if err != nil {
		return fmt.Errorf("PKCS7 去填充失败: %w", err)
	}

	// 切割掉前面的 "neteasecloudmusic"
	if !bytes.HasPrefix(unpaddedData, prefixToRemove) {
		return fmt.Errorf("非法数据：未检测到标准文件头前缀")
	}
	d.Rc4Key = unpaddedData[len(prefixToRemove):]

	return nil
}

// 工具函数：AES-ECB 解密 (Go 标准库未直接提供 ECB，需手动按块迭代)
func aesECBDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	if len(ciphertext)%blockSize != 0 {
		return nil, fmt.Errorf("密文长度必须是 BlockSize (%d) 的倍数", blockSize)
	}

	plaintext := make([]byte, len(ciphertext))
	for start := 0; start < len(ciphertext); start += blockSize {
		end := start + blockSize
		block.Decrypt(plaintext[start:end], ciphertext[start:end])
	}

	return plaintext, nil
}

// 工具函数：PKCS7 去填充
func pkcs7Unpad(src []byte) ([]byte, error) {
	length := len(src)
	if length == 0 {
		return nil, fmt.Errorf("数据为空，无法去填充")
	}

	// 获取最后一个字节，其数值代表填充的字节数
	paddingLen := int(src[length-1])

	// 验证填充的合法性
	if paddingLen < 1 || paddingLen > 16 || paddingLen > length {
		return nil, fmt.Errorf("不合法的填充长度")
	}

	// 校验所有填充字节是否都等于 paddingLen
	for i := length - paddingLen; i < length; i++ {
		if src[i] != byte(paddingLen) {
			return nil, fmt.Errorf("PKCS7 填充损坏")
		}
	}

	return src[:length-paddingLen], nil
}
