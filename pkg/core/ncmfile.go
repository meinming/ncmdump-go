package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"ncmdump/pkg/crypto"
	"ncmdump/pkg/logger"
	"net/http"
	"os"
)

type NeteaseCloudMusicFile struct {
	Path        string
	file        *os.File
	Rc4Key      []byte
	CoverPic    []byte
	Metadata    *crypto.NcmMetadata
	MusicStream []byte
}

func NewNeteaseCloudMusicFile(path string) *NeteaseCloudMusicFile {
	return &NeteaseCloudMusicFile{Path: path}
}

func (n *NeteaseCloudMusicFile) Decrypt() error {
	file, err := os.Open(n.Path)
	if err != nil {
		return fmt.Errorf("Failed to read file:%w", err)
	}
	n.file = file
	defer n.file.Close()

	// Magic Header Check
	headerBytes := make([]byte, 8)
	_, err = n.file.Read(headerBytes)
	if err != nil {
		return fmt.Errorf("Failed to read Header: %w", err)
	}
	if string(headerBytes) != "CTENFDAM" {
		return fmt.Errorf("Not a valid NetEase Cloud Music (NCM) file.")
	}
	logger.Debug("NCM Verify is sucessed.")

	_, _ = n.file.Seek(2, 1) // 移动2位

	lengthBytes := make([]byte, 4)
	_, err = n.file.Read(lengthBytes)
	if err != nil {
		return fmt.Errorf("Failed to read Length of AES key: %w", err)
	}
	keyLength := binary.LittleEndian.Uint32(lengthBytes)
	logger.Debug("Reading length of AES key is succeed: %v bytes", keyLength)

	keyCiphertext := make([]byte, keyLength)
	_, err = n.file.Read(keyCiphertext)
	if err != nil {
		return fmt.Errorf("Failed to read key cipher: %w", err)
	}

	h := crypto.Decoder{Rc4KeyEnc: keyCiphertext}

	err = h.DecryptRC4Key()
	if err != nil {
		return fmt.Errorf("Failed to decrypt RC4 key with AES: %w", err)
	}
	n.Rc4Key = h.Rc4Key

	metaLengthBytes := make([]byte, 4)
	_, err = n.file.Read(metaLengthBytes)
	if err != nil {
		return fmt.Errorf("Failed to read metadata length: %w", err)
	}
	h.MetadataEncSize = binary.LittleEndian.Uint32(metaLengthBytes)
	logger.Debug("Reading length of metadata is succeed: %v bytes", h.MetadataEncSize)
	metadataEnc := make([]byte, h.MetadataEncSize)
	_, err = n.file.Read(metadataEnc)
	if err != nil {
		return fmt.Errorf("Failed to read metadata cipher: %w", err)
	}
	h.MetadataEnc = metadataEnc
	err = h.DecryptMetadata()
	if err != nil {
		return fmt.Errorf("Failed to decrypt metadata: %w", err)
	}
	n.Metadata = h.Metadata

	_, _ = n.file.Seek(5+4, 1) // 跳过 5 字节 gap + 4 字节 cover crc = 9 字节

	coverLengthBytes := make([]byte, 4)
	_, err = n.file.Read(coverLengthBytes)
	if err != nil {
		return fmt.Errorf("Failed to read Length of cover: %w", err)
	}
	coverLength := binary.LittleEndian.Uint32(coverLengthBytes)
	logger.Debug("Reading length of cover is succeed: %d bytes", coverLength)
	switch n.Metadata.Format {
	case "flac":
		if coverLength != 0 {
			logger.Warn("存在Cover, 可能出现问题")
			CoverPic := make([]byte, coverLength)
			_, err = n.file.Read(CoverPic)
			// os.WriteFile("test.png", CoverPic, 0664)

			if err != nil {
				return fmt.Errorf("读取 FLAC 本地封面字节失败: %w", err)
			}
			n.CoverPic = CoverPic
		} else if n.Metadata.AlbumPic != "" {
			n.CoverPic, err = coverGet(n.Metadata.AlbumPic)
			if err != nil {
				logger.Warn("从网络下载 FLAC 封面失败: %v，将生成无封面音频", err)
			}
		}
	case "mp3":
		if coverLength > 0 {
			CoverPic := make([]byte, coverLength)
			_, err = n.file.Read(CoverPic)
			// os.WriteFile("test.png", CoverPic, 0664)

			if err != nil {
				return fmt.Errorf("读取 MP3 本地封面字节失败: %w", err)
			}
			n.CoverPic = CoverPic
		} else {
			logger.Warn("该 ncm 文件未包含内置封面(?) 可能引发错误")
		}
	}

	var musicStream bytes.Buffer
	buf := make([]byte, 32*1024)
	ncmRc4 := crypto.NewNCMRC4(n.Rc4Key)
	logger.Debug("正在解密流媒体：%v", n.Metadata.MusicName)
	for {
		r, err := n.file.Read(buf)
		if r > 0 {
			currentBlock := buf[:r]
			ncmRc4.Decrypt(currentBlock)

			_, writeErr := musicStream.Write(currentBlock)
			if writeErr != nil {
				return fmt.Errorf("写入二进制流失败: %w", writeErr)
			}
		}

		if err == io.EOF {
			break // 文件读完了，安全退出
		}
		if err != nil {
			return fmt.Errorf("读取音频数据失败: %w", err)
		}
	}
	n.MusicStream = musicStream.Bytes()

	return nil
}

func coverGet(imgUrl string) ([]byte, error) {
	logger.Debug("正在请求封面：%v", imgUrl)
	resp, err := http.Get(imgUrl)
	if err != nil {
		return nil, fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取图片二进制流失败: %w", err)
	}

	return imgBytes, nil
}
