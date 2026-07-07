package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"ncmdump/pkg/logger"
)

var aesKeyMetadata = []byte{
	0x23, 0x31, 0x34, 0x6C, 0x6A, 0x6B, 0x5F, 0x21,
	0x5C, 0x5D, 0x26, 0x30, 0x55, 0x3C, 0x27, 0x28,
} // "2331346C6A6B5F215C5D2630553C2728"

const metadataXorByte byte = 0x63

var metaPrefixToRemove = []byte("163 key(Don't modify):")

// NcmMetadata 解密后 JSON 映射的结构体
type NcmMetadata struct {
	Format    string       `json:"format"`    //"flac"
	MusicName string       `json:"musicName"` //"カタオモイ",
	Artist    []ArtistInfo `json:"artist"`    //[["Aimer", 16152]],
	Album     string       `json:"album"`     //"daydream",
	AlbumPic  string       `json:"albumPic"`  //http://p1.music.126.net/2QRYxUqXfW0zQpm2_DVYRA==/109951165052089697.jpg"
	// musicId int64          //431259256,
	// albumId	int64		  // 34826361,
	// albumPicDocId int64    // 109951165052089697,
	// mvId:                  //0
	// flag                   // 4
	// bitrate                // 876923,
	// duration               //v207866,
	// alias  []String        //[]
	// transNames             //["单相思"]
}

// ArtistInfo 定义单一歌手的嵌套结构
type ArtistInfo struct {
	Name string // 歌手名字
	ID   int64  // 歌手ID
}

func (m *NcmMetadata) GetArtists() []string {
	var artists []string
	for _, artist := range m.Artist {
		artists = append(artists, artist.Name)
	}
	if len(artists) == 0 {
		return []string{"Unknown"}
	}
	return artists
}

// DecryptMetadata 执行元数据解密与反序列化
func (d *Decoder) DecryptMetadata() error {
	// 如果加密数据大小为 0，直接跳过
	if d.MetadataEncSize == 0 {
		return nil
	}

	// 1. 异或混淆处理
	xorData := make([]byte, len(d.MetadataEnc))
	for i, b := range d.MetadataEnc {
		xorData[i] = b ^ metadataXorByte
	}

	// 2. 校验并裁剪前缀 "163 key(Don't modify):"
	if !bytes.HasPrefix(xorData, metaPrefixToRemove) {
		return fmt.Errorf("非法元数据：未检测到合法的元数据前缀")
	}
	base64Data := xorData[len(metaPrefixToRemove):]

	// 3. Base64 解码
	decodedData := make([]byte, base64.StdEncoding.DecodedLen(len(base64Data)))
	n, err := base64.StdEncoding.Decode(decodedData, base64Data)
	if err != nil {
		return fmt.Errorf("元数据 Base64 解码失败: %w", err)
	}
	decodedData = decodedData[:n] // 截取到实际解码长度

	// 4. AES-ECB 解密
	decryptedData, err := aesECBDecrypt(decodedData, aesKeyMetadata)
	if err != nil {
		return fmt.Errorf("元数据 AES 解密失败: %w", err)
	}

	// 5. 去除 PKCS7 填充
	unpaddedData, err := pkcs7Unpad(decryptedData)
	if err != nil {
		return fmt.Errorf("元数据 PKCS7 去填充失败: %w", err)
	}

	// 6. 将裁剪掉前缀 "music:" 后的 JSON 明文解析到结构体中
	// 解密出的明文通常形如: music:{"musicId":444269135,"musicName":"..."}
	musicPrefix := []byte("music:")
	djPrefix := []byte("dj:")
	unpaddedData = bytes.TrimPrefix(unpaddedData, musicPrefix)
	unpaddedData = bytes.TrimPrefix(unpaddedData, djPrefix)
	logger.Debug("metadeta: %v", string(unpaddedData))

	var meta NcmMetadata
	// json.Unmarshal 相当于将 JSON 字符串反序列化为 Go 的对象
	if err := json.Unmarshal(unpaddedData, &meta); err != nil {
		return fmt.Errorf("元数据 JSON 解析失败: %w", err)
	}
	d.Metadata = &meta
	return nil
}

func (a *ArtistInfo) UnmarshalJSON(data []byte) error {
	// 1. 先用一个空接口切片接收这个数组：[ "周杰伦", 6452 ]
	var raw []interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// 2. 严谨校验长度（至少包含名字和ID）
	if len(raw) < 2 {
		return fmt.Errorf("invalid artist format")
	}

	// 3. 断言并提取第一个元素（名字）
	if name, ok := raw[0].(string); ok {
		a.Name = name
	}

	// 4. 断言并提取第二个元素（ID）。JSON 中的数字默认会解析为 float64
	if id, ok := raw[1].(float64); ok {
		a.ID = int64(id)
	}

	return nil
}

/*
	Metadata for ncm file.

    `music`:

    ```json
    {
        "format": "flac",
        "musicId": 431259256,
        "musicName": "カタオモイ",
        "artist": [["Aimer", 16152]],
        "album": "daydream",
        "albumId": 34826361,
        "albumPicDocId": 109951165052089697,
        "albumPic": "http://p1.music.126.net/2QRYxUqXfW0zQpm2_DVYRA==/109951165052089697.jpg",
        "mvId": 0,
        "flag": 4,
        "bitrate": 876923,
        "duration": 207866,
        "alias": [],
        "transNames": ["单相思"]
    }
    ```

    `dj`:

    ```json
    {
        "programId": 2506516081,
        "programName": "03 踏遍万水千山",
        "mainMusic": {
            "musicId": 1957438579,
            "musicName": "03 踏遍万水千山",
            "artist": [],
            "album": "[DJ节目]北方文艺出版社的DJ节目 第8期",
            "albumId": 0,
            "albumPicDocId": 109951167551086981,
            "albumPic": "https://p1.music.126.net/M48NPuT591tIqqUdQyKZlg==/109951167551086981.jpg",
            "mvId": 0,
            "flag": 0,
            "bitrate": 320000,
            "duration": 1222948,
            "alias": [],
            "transNames": []
        },
        "djId": 7891086863,
        "djName": "北方文艺出版社",
        "djAvatarUrl": "http://p1.music.126.net/DQr2q_S23tYY8vU_C-kAYw==/109951167535553901.jpg",
        "createTime": 1655691020376,
        "brand": "林徽因传：倾我所能去坚强",
        "serial": 3,
        "programDesc": "这是一本有温度、有态度的传记，记录了真正意义上的民国女神——林徽因，从容坚强、传奇丰沛的一生。",
        "programFeeType": 15,
        "programBuyed": true,
        "radioId": 977264730,
        "radioName": "林徽因传：倾我所能去坚强",
        "radioCategory": "文学出版",
        "radioCategoryId": 3148096,
        "radioDesc": "这是一本有温度、有态度的传记，记录了真正意义上的民国女神——林徽因，从容坚强、传奇丰沛的一生。",
        "radioFeeType": 1,
        "radioFeeScope": 0,
        "radioBuyed": true,
        "radioPrice": 30,
        "radioPurchaseCount": 0
    }
*/
