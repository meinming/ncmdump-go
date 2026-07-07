# NCMDUMP-GO

🚀 **高效、健壮的 .ncm 还原工具**

本项目基于 Go 语言重构，旨在探索密码学原理并在 Go CLI 工程中付诸实践。工具具备较强的容错与批量处理能力，支持自动从内嵌数据或网络 API 中提取并还原无损 FLAC 与 MP3 的媒体标签（元数据及专辑封面）。

---

## ✨ 核心特性

- **现代算法重构**：利用底层硬件溢出机制替代传统的位运算取模（`& 0xFF`），提供更优的解密响应速度。
- **现代工业级组件**：基于 Cobra 框架重构系统底座，遵循工业级标准规范 CLI 命令与参数调度。
- **全格式完美兼容**：无缝支持包含内置图片的早期 MP3/FLAC NCM，以及无内置封面但自带网络 URL 的现代化 NCM 音频结构。
- **全自动化媒体封装**：自动提取歌曲名、歌手、专辑并注入 ID3v2 (MP3) 或 Vorbis Comment (FLAC) 容器，完整保留无损视听体验。

## 🛠️ 快速开始

### 1. 克隆与编译
确保您的开发环境已安装 **Go 1.22+**：

```bash
git clone [https://github.com/meinming/ncmdump-go.git](https://github.com/meinming/ncmdump-go.git)
cd ncmdump-go
go build -o ncmdump main.go
```

### 2. 基础命令行使用

**批量转换指定文件(目录)：**

```bash
./ncmdump song1.ncm song2.ncm ./test
```

**指定输入与输出目录：**

```bash
./ncmdump -i ./music_encrypted -o ./music_decrypted
```

**高级选项（不写入元数据及封面）：**

```bash
./ncmdump --no-metadata --no-cover song.ncm
```

### 3. 支持的命令行参数

| 参数短写 | 参数全称 | 默认值 | 描述 |
| --- | --- | --- | --- |
| `-i` | `--input` | `.` | 指定输入的包含 `.ncm` 的文件夹路径 | 
| `-o` | `--output` | `.` | 指定还原后的音频文件输出目录 |
| `-t` | `--traversal` | `false` | 启动遍历模式，自动遍历文件夹下所有文件 |
| `-d` | `--debug` | `false` | 启用调试模式，输出详细解密细节与流状态 |
|  | `--no-metadata` | `false` | 仅解密原始音频，不注入歌曲标签（如歌名、歌手等） |
|  | `--no-cover` | `false` | 解密音频时跳过专辑封面注入 |

## 📐 系统架构与解密原理

工具的数据控制流遵循严格的二进制空间划分：

```text
[.ncm 骨架结构] 
  ├── 1. 魔法头校验 ───> 匹配 "CTENFDAM" (8 字节)
  ├── 2. 核心密钥提取 ──> 常量 AES-ECB 解密 ──> 生成魔改 RC4 密码箱
  ├── 3. 元数据解析 ───> JSON 异步反序列化 (提取 format/歌曲信息)
  ├── 4. 封面流提取 ───> 优先抽取本地嵌样 (Length > 0) ──> 备用网络 API 补全
  └── 5. 异或流解密 ───> 32KB 分块并发缓冲 ──> 音频容器封装 (ID3v2/Vorbis)
```

## 📝 许可证

本项目遵循 [Apache-2.0 开源许可证](https://www.apache.org/licenses/LICENSE-2.0.html) 授权。本工具仅用于学术研究与个人技术探讨，开发者不承担任何因不当使用或商业化衍生所导致的法律责任。

## 🥰 特别鸣谢

- 感谢我亦师亦友的朋友@eternal-flame-AD提供的支持
- 感谢同类项目：[yoki/ncmdump](https://github.com/yoki123/ncmdump)
