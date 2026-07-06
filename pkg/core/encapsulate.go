package core

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // 注册 JPEG 自动解码器
	_ "image/png"  // 注册 PNG 自动解码器
	"ncmdump/pkg/logger"
	"os"
	"strings"

	// 引入第三方音频封装库
	"github.com/bogem/id3v2"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

// 核心方法 1：MP3 容器封装 (ID3v2 标准)
func (n *NeteaseCloudMusicFile) EncapsulateMp3(outputPath string, noMeta bool, noCv bool) error {
	// 1. 先将解密出来的裸音频数据流（MusicStream）写入目标文件
	err := os.WriteFile(outputPath, n.MusicStream, 0644)
	if err != nil {
		return fmt.Errorf("写入 MP3 基础音频流失败: %w", err)
	}

	// 2. 打开该 MP3 文件开始注入 ID3 标签
	tag, err := id3v2.Open(outputPath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("打开 MP3 ID3 容器失败: %w", err)
	}
	defer tag.Close()

	// 设置全局编码为 UTF-8，防止在老旧播放器或 Windows 上中文乱码
	tag.SetDefaultEncoding(id3v2.EncodingUTF8)
	tag.DeleteAllFrames()

	if noMeta && noCv {
		return tag.Save()
	}

	// 3. 填入文本标签 (TIT2=歌名, TALB=专辑, TPE1=歌手)
	tag.AddTextFrame("TIT2", id3v2.EncodingUTF8, n.Metadata.MusicName)
	tag.AddTextFrame("TALB", id3v2.EncodingUTF8, n.Metadata.Album)

	// 多个歌手用斜杠 "/" 拼接
	artistsStr := strings.Join(n.Metadata.GetArtists(), "/")
	tag.AddTextFrame("TPE1", id3v2.EncodingUTF8, artistsStr)
	logger.Debug("添加元数据：TIT2=%v，TALB=%v，TPE1=%v", n.Metadata.MusicName, n.Metadata.Album, artistsStr)

	// 4. 填入封面标签 (APIC 帧)
	if len(n.CoverPic) > 0 && !noCv {

		// 动态识别封面是 image/jpeg 还是 image/png
		_, imgType, err := image.DecodeConfig(bytes.NewReader(n.CoverPic))
		mimeType := "image/jpeg"
		if err == nil && imgType != "" {
			mimeType = "image/" + imgType
		}
		logger.Debug("开始添加封面：minmeType=%v(%v)", mimeType, imgType)
		picFrame := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mimeType,
			PictureType: id3v2.PTFrontCover, // 正面封面标记
			Description: "Front cover",
			Picture:     n.CoverPic,
		}
		tag.AddAttachedPicture(picFrame)
		logger.Debug("成功添加封面")
	}

	// 5. 保存并闭合文件
	return tag.Save()
}

// 核心方法 2：FLAC 容器封装 (Metadata Block 标准)
func (n *NeteaseCloudMusicFile) EncapsulateFlac(outputPath string, noMeta bool, noCv bool) error {
	// 1. 先将解密出来的无损数据流写入目标文件
	err := os.WriteFile(outputPath, n.MusicStream, 0644)
	if err != nil {
		return fmt.Errorf("写入 FLAC 基础音频流失败: %w", err)
	}

	// 2. 解析刚写入的 FLAC 二进制骨架
	f, err := flac.ParseFile(outputPath)
	if err != nil {
		return fmt.Errorf("解析 FLAC 容器失败: %w", err)
	}

	// 3. 组装文本标签块 (Vorbis Comment)
	var commentBlock *flac.MetaDataBlock
	// 尝试寻找现有的评论块，找不到就新建一个
	for _, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			commentBlock = block
			break
		}
	}

	var vc *flacvorbis.MetaDataBlockVorbisComment
	if commentBlock != nil {
		vc, err = flacvorbis.ParseFromMetaDataBlock(*commentBlock)
		if err != nil {
			return err
		}
	} else {
		vc = flacvorbis.New()
	}

	// 写入键值对标签
	_ = vc.Add("TITLE", n.Metadata.MusicName)
	_ = vc.Add("ALBUM", n.Metadata.Album)
	for _, artist := range n.Metadata.GetArtists() {
		_ = vc.Add("ARTIST", artist)
	}

	// 把修改后的评论块放回 FLAC 结构中
	updatedComment := vc.Marshal()
	if commentBlock != nil {
		*commentBlock = updatedComment
	} else {
		f.Meta = append(f.Meta, &updatedComment)
	}

	// 4. 组装复杂的图片标签块 (METADATA_BLOCK_PICTURE)
	if len(n.CoverPic) > 0 {
		// 计算机思维：用流读取图片头部的宽高，而不需要在内存中完全把图片解压放大
		_, imgType, err := image.DecodeConfig(bytes.NewReader(n.CoverPic))
		if err != nil {
			return fmt.Errorf("解析封面图片维度失败: %w", err)
		}
		mimeType := "image/jpeg"
		if imgType != "" {
			mimeType = "image/" + imgType
		}

		// 构建 FLAC 规范的 Picture 块
		fp, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Cover", n.CoverPic, mimeType)
		if err != nil {
			return fmt.Errorf("")
		}

		// 序列化为二进制数据块并追加进 FLAC
		pictureBlock := fp.Marshal()
		f.Meta = append(f.Meta, &pictureBlock)
	}

	// 5. 将打好全部元数据补丁的 FLAC 结构重新存盘
	return f.Save(outputPath)
}
