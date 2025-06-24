package compile

import (
	"archive/tar"
	"backend/pkg/parse"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type ToolChain struct {
	Name    string
	Type    string
	Version string
	Path    string
	CC      string
	LIB     string
}

// KernelURL kernel download URL prefix
const KernelURL = "https://github.com/torvalds/linux/archive/"

// GlobalToolChain global toolchain for certain bug construct. Use InitToolChain to decide toolchain automatically
var GlobalToolChain *ToolChain = nil

// if deploy locally, please change below to adapt your environment
var toolChains = map[string]string{
	"gcc-10.2.0": "toolchain/gcc/gcc-10.2.0",
	"gcc-10.1.0": "toolchain/gcc/gcc-10.1.0",
	"gcc-15.1.0": "toolchain/gcc/gcc-15.1.0",
	"gcc-9.1.0":  "toolchain/gcc/gcc-9.1.0",
}

// necessary configs for kernel debug and memory check ; include sysrq-trigger
var requiredConfigs = map[string]string{
	"CONFIG_KEXEC":                    "y",
	"CONFIG_KEXEC_FILE":               "y",
	"CONFIG_CRASH_DUMP":               "y",
	"CONFIG_RELOCATABLE":              "y",
	"CONFIG_BLK_DEV_INITRD":           "y",
	"CONFIG_DEVTMPFS":                 "y",
	"CONFIG_DEVTMPFS_MOUNT":           "y",
	"CONFIG_DEBUG_INFO":               "y",
	"CONFIG_PROC_VMCORE":              "y",
	"CONFIG_VT":                       "y",
	"CONFIG_CONSOLE_TRANSLATIONS":     "y",
	"CONFIG_FB":                       "y",
	"CONFIG_SYSFS":                    "y",
	"CONFIG_SYSRQ":                    "y",
	"CONFIG_KASAN":                    "y",
	"CONFIG_KASAN_GENERIC":            "y",
	"CONFIG_KASAN_INLINE":             "y",
	"CONFIG_DEBUG_KMEMLEAK":           "y",
	"CONFIG_MAGIC_SYSRQ":              "y",
	"CONFIG_DEBUG_KMEMLEAK_AUTO_SCAN": "y",
	"CONFIG_GDB_SCRIPTS":              "y",
}

// config environs for kernel build; use Toolchain build manually
func buildEnv() []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("PATH=%s:%s", GlobalToolChain.Path, os.Getenv("PATH")))
	env = append(env, fmt.Sprintf("CC=%s", GlobalToolChain.CC))
	env = append(env, fmt.Sprintf("HOSTCC=%s", GlobalToolChain.CC))
	env = append(env, fmt.Sprintf("AR=%s", fmt.Sprintf("%s/gcc-ar", GlobalToolChain.Path)))
	return env
}

// kernelPath Construct directory with CrashReport
func kernelPath(report *parse.CrashReport) string {
	return fmt.Sprintf("build/%s/linux-%s", report.Crashes[0].KernelSourceCommit, report.Crashes[0].KernelSourceCommit)
}

// fileExists check if one file exist
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// dirExistsAndNotEmpty
func dirExistsAndNotEmpty(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("path %s is not a directory", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	return len(entries) > 0, nil
}

// DownloadKernel download kernel archive
func DownloadKernel(report *parse.CrashReport) error {
	if GlobalToolChain == nil {
		return fmt.Errorf("toolchain not initialized")
	}

	if len(report.Crashes) == 0 {
		return fmt.Errorf("no valid report data")
	}

	commit := report.Crashes[0].KernelSourceCommit
	if commit == "" {
		return fmt.Errorf("no valid kernel commit")
	}

	downloadURL := KernelURL + commit + ".tar.gz"
	fileName := "linux-" + commit + ".tar.gz"

	saveDir := "build/" + commit
	err := os.MkdirAll(saveDir, 0755)
	if err != nil {
		return err
	}

	tarFilePath := filepath.Join(saveDir, fileName)
	sourceDir := filepath.Join(saveDir, "linux-"+commit)

	log.Infoln("download dir:", downloadURL)
	log.Infoln("save as:", tarFilePath)
	log.Infoln("decompress dir:", sourceDir)

	if fileExists(tarFilePath) {
		log.Infoln("file already exists, skip download", fileName)
	} else {
		resp, err := http.Get(downloadURL)
		if err != nil {
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorln(err)
			}
		}()

		outFile, err := os.Create(tarFilePath)
		if err != nil {
			return err
		}
		defer func() {
			if err := outFile.Close(); err != nil {
				log.Errorln(err)
			}
		}()

		//bar := progressbar.NewOptions(-1, progressbar.OptionSetDescription("downloading"))

		_, err = io.Copy(io.Writer(outFile), resp.Body)
		if err != nil {
			return err
		}
		//err = bar.Finish()
		//if err != nil {
		//	return err
		//}
		log.Infoln("\ndownload linux tar success")
	}

	exists, err := dirExistsAndNotEmpty(sourceDir)
	if err != nil {
		return err
	}
	if exists {
		log.Infoln("target dir already exists, skip decompress", fileName)
	} else {
		f, err := os.Open(tarFilePath)
		if err != nil {
			return err
		}

		defer func() {
			if err := f.Close(); err != nil {
				log.Errorln(err)
			}
		}()

		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer func() {
			if err := gzReader.Close(); err != nil {
				log.Errorln(err)
			}
		}()

		tarReader := tar.NewReader(gzReader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				return err
			}

			target := filepath.Join(saveDir, header.Name)
			if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(saveDir)+string(os.PathSeparator)) {
				return fmt.Errorf("illegal file path in archive: %s", target)
			}

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}
			case tar.TypeReg:
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					err := outFile.Close()
					if err != nil {
						return err
					}
					return err
				}
				if err := outFile.Close(); err != nil {
					return err
				}
			case tar.TypeSymlink:
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				if err := os.Symlink(header.Linkname, target); err != nil {
					return fmt.Errorf("failed create symbol link: %s -> %s (%v)", target, header.Linkname, err)
				}
			case tar.TypeLink:
				linkTarget := filepath.Join(saveDir, header.Linkname)
				if err := os.Link(linkTarget, target); err != nil {
					return fmt.Errorf("failed create hard symbol link: %s -> %s (%v)", target, linkTarget, err)
				}
			default:
				log.Debugf("skip type (not support): %s (%c)\n", header.Name, header.Typeflag)
			}
		}
		log.Infoln("decompress tar success, linux kernel exists", sourceDir)
	}

	return nil
}

// DownloadBug download bug reproducer file (c file)
func DownloadBug(report *parse.CrashReport) error {
	if GlobalToolChain == nil {
		return errors.New("toolchain not initialized")
	}

	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
	}

	reproducerURL := parse.SyzkallerURL + report.Crashes[0].CReproducer
	log.Infof("downloading c reproducer from %s", reproducerURL)
	buildDir := fmt.Sprintf("build/%s/%s", report.Crashes[0].KernelSourceCommit, fmt.Sprintf("linux-%s", report.Crashes[0].KernelSourceCommit))
	reproducerFile := fmt.Sprintf("%s/bug.c", buildDir)

	resp, err := client.Get(reproducerURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	//total := resp.ContentLength

	outFile, err := os.Create(reproducerFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	//var bar *progressbar.ProgressBar
	//if total < 0 {
	//	bar = progressbar.NewOptions(-1, progressbar.OptionSetDescription("downloading"))
	//} else {
	//	bar = progressbar.DefaultBytes(
	//		total,
	//		"downloading",
	//	)
	//}

	_, err = io.Copy(io.Writer(outFile), resp.Body)
	if err != nil {
		return err
	}

	//err = bar.Finish()
	//if err != nil {
	//	return err
	//}

	fmt.Println("")
	log.Infoln("download c reproducer success")

	return nil
}

// DownloadConfig download config form syzkaller
func DownloadConfig(report *parse.CrashReport) error {
	if GlobalToolChain == nil {
		return errors.New("toolchain not initialized")
	}

	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
	}

	configURL := parse.SyzkallerURL + report.Crashes[0].KernelConfig
	log.Infof("downloading kernel config from %s", configURL)
	buildDir := fmt.Sprintf("build/%s/%s", report.Crashes[0].KernelSourceCommit, fmt.Sprintf("linux-%s", report.Crashes[0].KernelSourceCommit))
	configFile := fmt.Sprintf("%s/.config", buildDir)

	if fileExists(configFile) {
		log.Infoln("file already exists, skip download", configFile)
	} else {
		resp, err := client.Get(configURL)
		if err != nil {
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorln(err)
			}
		}()

		//total := resp.ContentLength

		outFile, err := os.Create(configFile)
		if err != nil {
			return err
		}
		defer func() {
			if err := outFile.Close(); err != nil {
				log.Error(err)
			}
		}()

		//var bar *progressbar.ProgressBar
		//if total < 0 {
		//	bar = progressbar.NewOptions(-1, progressbar.OptionSetDescription("downloading"))
		//} else {
		//	bar = progressbar.DefaultBytes(
		//		total,
		//		"downloading",
		//	)
		//}

		_, err = io.Copy(io.Writer(outFile), resp.Body)
		if err != nil {
			return err
		}

		//err = bar.Finish()
		//if err != nil {
		//	return err
		//}
		fmt.Println("")
		log.Infoln("download linux config success")
	}

	err := checkFix(configFile)
	if err != nil {
		return err
	}

	return nil
}

// checkFix check config of kernel if required all satisfied. if not, try fix it
func checkFix(configPath string) error {
	f, err := os.Open(configPath)
	if err != nil {
		log.Error(err)
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorln(err)
		}
	}()

	var lines []string
	configMap := make(map[string]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		configMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	log.Infof("using config file: %s\n", configPath)
	log.Infoln("checking kernel config for kdump")

	flag := true
	found := make(map[string]bool)

	for i, line := range lines {
		for key, expected := range requiredConfigs {
			if strings.HasPrefix(line, key+"=") || strings.HasPrefix(line, "# "+key+" is not set") {
				actual := configMap[key]
				if actual != expected {
					log.Infof("[✘] error config: %s (expected: %s)\n", key, expected)
					lines[i] = fmt.Sprintf("%s=%s", key, expected)
					flag = false
				} else {
					log.Infof("[✔] %s=%s\n", key, expected)
				}
				found[key] = true
			}
		}
	}

	if !flag {
		out, err := os.Create(configPath)
		if err != nil {
			return err
		}
		defer func() {
			if err := out.Close(); err != nil {
				log.Errorln(err)
			}
		}()

		writer := bufio.NewWriter(out)
		for _, line := range lines {
			_, _ = writer.WriteString(line + "\n")
		}
		err = writer.Flush()
		if err != nil {
			return err
		}

		log.Infoln("config updated. running \"make olddefconfig\"")

		cmd := exec.Command("make", fmt.Sprintf("CC=%s", GlobalToolChain.CC), fmt.Sprintf("HOSTCC=%s", GlobalToolChain.CC), "olddefconfig")
		cmd.Dir = filepath.Dir(configPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed execute \"make olddefconfig\": %v", err)
		}
	}

	return nil
}

// configCompiler parse compiler used in crash report
func configCompiler(report *parse.CrashReport) (string, string) {
	desc := report.Crashes[0].CompilerDescription
	re := regexp.MustCompile(`(?i)(gcc|clang).*?(\d+\.\d+\.\d+)`)
	matches := re.FindAllStringSubmatch(desc, -1)
	var compiler string
	var version string
	for _, match := range matches {
		compiler = strings.ToLower(match[1])
		version = match[2]
	}
	return compiler, version
}

func parseVersion(version string) (int, int, int) {
	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	match := re.FindStringSubmatch(version)
	if len(match) != 4 {
		return -1, -1, -1
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	return major, minor, patch
}

func abs(n int) int {
	if n < 0 {
		return n * -1
	}
	return n
}

func findToolchain(compiler string, version string) (string, error) {
	vMajor, vMinor, vPatch := parseVersion(version)
	type kv struct {
		key   string
		path  string
		major int
		diff1 int
		diff2 int
	}
	var candidates []kv
	for k, path := range toolChains {
		if !strings.HasPrefix(k, compiler+"-") {
			continue
		}
		kMajor, kMinor, kPatch := parseVersion(k)
		if kMajor == vMajor {
			diff1 := abs(kMinor - vMinor)
			diff2 := abs(kPatch - vPatch)
			candidates = append(candidates, kv{k, path, kMajor, diff1, diff2})
		}
	}
	if len(candidates) == 0 {
		return "", errors.New("no toolchain found for " + compiler + "-" + version)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].diff1 == candidates[j].diff1 {
			return candidates[i].diff2 < candidates[j].diff2
		} else {
			return candidates[i].diff1 < candidates[j].diff1
		}
	})
	return candidates[0].key, nil
}

func setToolchain(report *parse.CrashReport) ToolChain {
	compiler, version := configCompiler(report)
	key, err := findToolchain(compiler, version)
	if err != nil {
		log.Panicf("Global Compiler Not Found: %s\n", err)
	}
	rootPath, _ := os.Getwd()
	return ToolChain{
		Name:    compiler + version,
		Type:    compiler,
		Version: version,
		Path:    filepath.Join(rootPath, toolChains[key]) + "/bin",
		CC:      filepath.Join(rootPath, toolChains[key]) + "/bin/gcc",
		LIB:     filepath.Join(rootPath, toolChains[key]) + "/lib64",
	}
}

// InitToolChain must be called once before any compile package function
func InitToolChain(report *parse.CrashReport) {
	tc := setToolchain(report)
	GlobalToolChain = &tc
}

func MakeKernel(report *parse.CrashReport) error {
	path := kernelPath(report)

	logger := log.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if GlobalToolChain == nil {
		return errors.New("toolchain not initialized")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	oldDir, err := os.Getwd()
	path = filepath.Join(oldDir, path)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			log.Errorln(err)
		}
	}()

	if err = os.Chdir(path); err != nil {
		return err
	}

	env := buildEnv()

	numCpu := runtime.NumCPU()
	makeJobs := fmt.Sprintf("-j%d", numCpu-1)

	configPath := filepath.Join(path, ".config")
	if !fileExists(configPath) {
		return errors.New("config file not found")
	}

	time.Sleep(time.Second)
	log.Infoln("starting kernel compilation in", path)

	compileCmd := exec.Command("bear", "--", "make", makeJobs)
	compileCmd.Env = env
	compileCmd.Dir = path

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	compileCmd.Stdout = io.MultiWriter(stdout, logger.Writer())
	compileCmd.Stderr = io.MultiWriter(stderr, logger.Writer())

	err = compileCmd.Run()
	if err != nil {
		return fmt.Errorf("error compiling kernel: %s", err)
	}

	log.Infoln("compilation succeeded")

	bzImagePath := filepath.Join(path, "arch/x86_64/boot/bzImage")
	if _, err := os.Stat(bzImagePath); os.IsNotExist(err) {
		return fmt.Errorf("bzImage file not found: %s", bzImagePath)
	}

	time.Sleep(time.Second)
	log.Infoln("starting linux header install", path)

	headerCmd := exec.Command("make", "headers_install", "INSTALL_HDR_PATH=./linux-header")
	headerCmd.Env = env
	headerCmd.Dir = path
	headerCmd.Stdout = io.MultiWriter(stdout, logger.Writer())
	headerCmd.Stderr = io.MultiWriter(stderr, logger.Writer())

	err = headerCmd.Run()
	if err != nil {
		return fmt.Errorf("error installing header: %s", err)
	}

	empty, err := dirExistsAndNotEmpty(filepath.Join(path, "linux-header"))
	if err != nil {
		return err
	}
	if !empty {
		return errors.New("linux header not generated")
	}

	return nil
}

func ClearCompile(report *parse.CrashReport) error {
	path := kernelPath(report)
	env := buildEnv()

	cleanCmd := exec.Command("make", "clean")
	cleanCmd.Env = env
	cleanCmd.Dir = path
	if err := cleanCmd.Run(); err != nil {
		return err
	}
	return nil
}

func ClearDownload(report *parse.CrashReport) error {
	// !TODO: clear resources
	// err = os.Remove(tarFilePath)
	// if err != nil {
	//	log.Error(err)
	// return err
	// }
	// log.Infoln("decompress linux tar success")
	return nil
}

func GeneratePatch(report *parse.CrashReport, patch string) error {
	return nil
}

func ApplyPatch(report *parse.CrashReport) error {
	return nil
}

func RebuildKernel(report *parse.CrashReport, patch string) error {
	return nil
}
