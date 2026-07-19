package core

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"

	"ncmdump/pkg/crypto"
	"ncmdump/pkg/logger"
)

type NeteaseCloudMusicFile struct {
	Path        string
	file        *os.File
	Rc4Key      []byte
	CoverPic    []byte
	Metadata    *crypto.NcmMetadata
	MusicStream io.ReadCloser
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
	fileConsumed := true
	defer func() {
		if fileConsumed {
			n.file.Close()
		}
	}()

	// Magic Header Check
	headerBytes := make([]byte, 8+6)
	_, err = io.ReadFull(n.file, headerBytes)
	if err != nil {
		return fmt.Errorf("Failed to read Header: %w", err)
	}
	if slices.Equal(headerBytes[:4], []byte{'f', 'L', 'a', 'C'}) {
		n.Metadata = &crypto.NcmMetadata{
			Format: "flac",
		}
		_, err = n.file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("Failed to seek file: %w", err)
		}
		fileConsumed = false
		n.MusicStream = n.file
		return nil
	}
	if slices.Equal(headerBytes[:3], []byte{0x49, 0x44, 0x33}) {
		n.Metadata = &crypto.NcmMetadata{
			Format: "mp3",
		}
		_, err = n.file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("Failed to seek file: %w", err)
		}
		fileConsumed = false
		n.MusicStream = n.file
		return nil
	}
	if string(headerBytes[:8]) != "CTENFDAM" {
		return fmt.Errorf("Not a valid NetEase Cloud Music (NCM) file.")
	}
	logger.Debug("NCM Verify is sucessed.")

	lengthBytes := headerBytes[8:]
	keyLength := binary.LittleEndian.Uint32(lengthBytes[2:])
	logger.Debug("Reading length of AES key is succeed: %v bytes", keyLength)

	keyCiphertextAndMetadataLength := make([]byte, keyLength+4)
	_, err = io.ReadFull(n.file, keyCiphertextAndMetadataLength)
	if err != nil {
		return fmt.Errorf("Failed to read key cipher: %w", err)
	}

	h := crypto.Decoder{Rc4KeyEnc: keyCiphertextAndMetadataLength[:keyLength]}

	err = h.DecryptRC4Key()
	if err != nil {
		return fmt.Errorf("Failed to decrypt RC4 key with AES: %w", err)
	}
	n.Rc4Key = h.Rc4Key

	h.MetadataEncSize = binary.LittleEndian.Uint32(keyCiphertextAndMetadataLength[keyLength:])
	logger.Debug("Reading length of metadata is succeed: %v bytes", h.MetadataEncSize)
	metadataEnc := make([]byte, h.MetadataEncSize+9+4)
	_, err = io.ReadFull(n.file, metadataEnc)
	if err != nil {
		return fmt.Errorf("Failed to read metadata cipher: %w", err)
	}
	h.MetadataEnc = metadataEnc[:h.MetadataEncSize]
	err = h.DecryptMetadata()
	if err != nil {
		return fmt.Errorf("Failed to decrypt metadata: %w", err)
	}
	n.Metadata = h.Metadata

	coverLength := binary.LittleEndian.Uint32(metadataEnc[h.MetadataEncSize+9:])
	logger.Debug("Reading length of cover is succeed: %d bytes", coverLength)

	switch n.Metadata.Format {
	case "flac":
		if coverLength != 0 {
			logger.Warn("存在Cover, 可能出现问题")
			n.CoverPic = make([]byte, coverLength)
			_, err = n.file.Read(n.CoverPic)
			// os.WriteFile("test.png", CoverPic, 0664)
			if err != nil {
				return fmt.Errorf("读取 FLAC 本地封面字节失败: %w", err)
			}
		} else if n.Metadata.AlbumPic != "" {
			n.CoverPic, err = coverGet(n.Metadata.AlbumPic)
			if err != nil {
				logger.Warn("从网络下载 FLAC 封面失败: %v，将生成无封面音频", err)
			}
		} else {
			logger.Warn("无法通过已知途径得到 FLAC 封面，将生成无封面音频")
		}
	case "mp3":
		if coverLength != 0 {
			n.CoverPic = make([]byte, coverLength)
			_, err = n.file.Read(n.CoverPic)
			// os.WriteFile("test.png", CoverPic, 0664)
			if err != nil {
				return fmt.Errorf("读取 MP3 本地封面字节失败: %w", err)
			}
		} else if n.Metadata.AlbumPic != "" {
			n.CoverPic, err = coverGet(n.Metadata.AlbumPic)
			if err != nil {
				logger.Warn("从网络下载 MP3 封面失败: %v，将生成无封面音频", err)
			}
		} else {
			logger.Warn("无法通过已知途径得到 MP3 封面，将生成无封面音频")
		}
	}

	fileConsumed = false
	n.MusicStream = crypto.NewNCMRC4Streaming(n.file, n.Rc4Key)

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
