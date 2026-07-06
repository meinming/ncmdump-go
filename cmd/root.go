package cmd

import (
	"ncmdump/pkg/core" // 引入核心解密包
	"ncmdump/pkg/logger"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	inFolder   string
	outFolder  string
	noMetadata bool
	noCover    bool
)

var debugMode bool

var rootCmd = &cobra.Command{
	Use:   "ncmdump [files...]",
	Short: "Decrypt .ncm files",
	Args:  cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debugMode {
			logger.DebugEnabled = true
			logger.Debug("调试模式已启动，开始捕获解密细节...")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		// 统一存储所有即将被处理的 .ncm 文件路径
		var targetFiles []string

		// 传入具体的文件列表 (位置参数)
		if len(args) > 0 {
			for _, file := range args {
				// 计算机思维：这里做一步安全校验，防止用户误输入非 ncm 文件
				if strings.HasSuffix(file, ".ncm") {
					targetFiles = append(targetFiles, file)
				} else {
					logger.Warn("[跳过] 文件 %s 后缀名不是 .ncm", file)
				}
			}
		}

		// 指定输入文件夹 (--in-folder)
		if inFolder != "" {
			// 读取文件夹下的所有文件和子目录
			files, err := os.ReadDir(inFolder)
			if err != nil {
				logger.Error("无法读取输入文件夹: %v", err)
			}

			for _, file := range files {
				// 排除子文件夹，且文件名必须以 .ncm 结尾
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".ncm") {
					// 将文件夹路径与文件名拼接成一个完整的绝对/相对路径
					fullPath := filepath.Join(inFolder, file.Name())
					targetFiles = append(targetFiles, fullPath)
				}
			}
		}

		// 如果无文件名，也没指定文件夹，就打印 `--help` 帮助文档
		if len(targetFiles) == 0 {
			_ = cmd.Help()
			return
		}

		// 打印初始化的解析状态（方便调试）
		logger.Info("开始处理，目标输出目录: %s", outFolder)
		logger.Debug("导出元数据: %v | 导出封面: %v", !noMetadata, !noCover)

		for _, ncmPath := range targetFiles {
			// 调用下方的具体执行函数
			processSingleFile(ncmPath, outFolder, noMetadata, noCover)
		}
	},
}

func init() {
	// Go 语言使用指针（&变量名）来让 Cobra 直接把命令行的数据写入到刚才定义的全局变量中。

	rootCmd.Flags().StringVar(&inFolder, "in-folder", "", "要批量解密的输入文件夹路径")
	rootCmd.Flags().StringVar(&outFolder, "out-folder", ".", "转换后音频文件的输出存放目录")

	rootCmd.Flags().BoolVar(&noMetadata, "no-metadata", false, "不携带歌曲的元数据")
	rootCmd.Flags().BoolVar(&noCover, "no-cover", false, "不额外携带歌曲的专辑封面图片")

	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "启用调试模式，输出详细日志")

	// 直接关闭控制台检查提示
	cobra.MousetrapHelpText = ""
}

// 4. 核心调度与业务执行
func processSingleFile(ncmPath string, outDir string, noMeta bool, noCv bool) {
	logger.Info("[正在处理]: %s ...", ncmPath)

	// 文件夹存在就不做操作，不存在就逐层创建
	err := os.MkdirAll(outDir, os.ModePerm)
	if err != nil {
		logger.Error("  [错误] 无法创建输出目录 %s: %v", outDir, err)
		return
	}

	fileName := filepath.Base(ncmPath)               // "周杰伦 - 晴天.ncm"
	baseName := strings.TrimSuffix(fileName, ".ncm") // "周杰伦 - 晴天"

	ncmFile := core.NewNeteaseCloudMusicFile(ncmPath)
	err = ncmFile.Decrypt()
	if err != nil {
		logger.Error("%v", err.Error())
	}
	audioFormat := ncmFile.Metadata.Format // "mp3" or "flac"

	musicOutputPath := filepath.Join(outDir, baseName+"."+audioFormat) // "./music_out/周杰伦 - 晴天.mp3"

	switch audioFormat {
	case "mp3":
		err := ncmFile.EncapsulateMp3(musicOutputPath, noMeta, noCv)
		if err != nil {
			logger.Error("%v", err.Error())
		}
	case "flac":
		err := ncmFile.EncapsulateFlac(musicOutputPath, noMeta, noCv)
		if err != nil {
			logger.Error("%v", err.Error())
		}
	}

	logger.Info("音频文件已成功还原至: %s", musicOutputPath)

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
}
