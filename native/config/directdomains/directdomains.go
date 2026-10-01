// Package directdomains 负责读取应用随配置目录下发的直连域名白名单。
//
// 单独成包而不是塞进 config：config 依赖 android 专用代码，主机上跑不了测试；
// 本包只依赖标准库与 mihomo 的日志，因此解析与校验规则可以被单元测试覆盖。
package directdomains

import (
	"os"
	P "path"
	"strings"

	"github.com/metacubex/mihomo/log"
)

// FileName 是白名单文件名，与 config.yaml 同目录。
//
// 走文件而不是编进二进制：内核是预编译产物，把域名烘进源码意味着每换一次自持域名
// 都要重新交叉编译 libclash.so。
const FileName = "direct-domains.txt"

// 与 app 侧校验保持一致的上限（DNS 名称总长 / 单个标签长度上限）。
const (
	MaxDomainLength      = 253
	MaxDomainLabelLength = 63
)

// Load 读取并规范化白名单。
//
// 语义约定：
//   - 文件缺失或不可读 = 无白名单，**不阻断**配置加载（app 侧清洗已注入主防线）；
//   - 非法条目逐条丢弃并记录日志，不影响其余条目；
//   - 大小写不敏感、自动去重；`#` 之后视为注释。
func Load(profileDir string) []string {
	data, err := os.ReadFile(P.Join(profileDir, FileName))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warnln("Read %s: %s", FileName, err.Error())
		}

		return nil
	}

	return Parse(string(data))
}

// Parse 解析白名单文本；与 Load 的规范化规则一致，但不涉及文件系统。
func Parse(data string) []string {
	domains := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)

	for _, line := range strings.Split(data, "\n") {
		domain := strings.ToLower(strings.TrimSpace(line))
		if idx := strings.IndexByte(domain, '#'); idx >= 0 {
			domain = strings.TrimSpace(domain[:idx])
		}
		if domain == "" {
			continue
		}
		if !IsValid(domain) {
			log.Warnln("Ignored invalid direct domain: %s", domain)

			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}

		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}

	return domains
}

// IsValid 与 app 侧 SanitizerInjector 的校验规则等价：
// 由点分隔的标签组成，字符集 [a-z0-9-_]，标签首尾必须是字母或数字。
func IsValid(domain string) bool {
	if len(domain) == 0 || len(domain) > MaxDomainLength {
		return false
	}

	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > MaxDomainLabelLength {
			return false
		}
		for i := 0; i < len(label); i++ {
			if !isChar(label[i]) {
				return false
			}
		}
		if !isEdgeChar(label[0]) || !isEdgeChar(label[len(label)-1]) {
			return false
		}
	}

	return true
}

func isChar(c byte) bool {
	return isEdgeChar(c) || c == '-' || c == '_'
}

func isEdgeChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}
