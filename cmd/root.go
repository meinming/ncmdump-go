package cmd

import (
	"fmt"
	"io/fs"
	"ncmdump/pkg/core" // 引入核心解密包
	"ncmdump/pkg/logger"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	inFolder    string
	outFolder   string
	noMetadata  bool
	noCover     bool
	scanSubDirs bool
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

		if len(args) > 0 {
			for _, file := range args {
				if isdir, err := IsDir(file); err == nil && isdir {
					dirFile, err := searchDir(file, scanSubDirs)
					targetFiles = append(targetFiles, dirFile...)
					if err != nil {
						logger.Error("%v", err)
					}
				} else if err != nil {
					logger.Error("%v", err)
				}
				// 安全校验，防止用户误输入非 ncm 文件
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
			dirFile, err := searchDir(inFolder, scanSubDirs)
			if err != nil {
				targetFiles = append(targetFiles, dirFile...)
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
	rootCmd.Flags().BoolVar(&noCover, "no-cover", false, "不携带歌曲的专辑封面图片")
	rootCmd.Flags().BoolVarP(&scanSubDirs, "traversal", "t", false, "启动遍历文件夹，自动遍历文件夹下所有文件")

	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "启用调试模式，输出详细日志")

	// 关闭控制台检查提示
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

func IsDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("路径不存在: %s", path)
		} else {
			return false, fmt.Errorf("发生其他错误: %w", err)
		}
	}

	// 3. 判断是目录还是文件
	if fileInfo.IsDir() {
		return true, nil
	} else {
		return false, nil
	}
}

func searchDir(root string, scanSubDirs bool) ([]string, error) {
	var Files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Warn("[跳过] 访问出错,已跳过目录%v: %v", path, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 如果当前路径不是根目录，并且是一个目录，且用户不想扫描子目录，则跳过
		if path != root && d.IsDir() && !scanSubDirs {
			return filepath.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(path, ".ncm") {
			Files = append(Files, path)
		}
		return nil
	})

	if err != nil {
		return []string{}, fmt.Errorf("扫描失败: %w", err)
	}
	return Files, nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
}
