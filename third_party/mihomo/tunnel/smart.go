package tunnel

import (
	"errors"
	"regexp"

	"github.com/metacubex/mihomo/component/loopback"
	"github.com/metacubex/mihomo/component/resolver"
	C "github.com/metacubex/mihomo/constant"
)

// countryCodeRegex 用于区分「GEOIP 规则 + 国家代码」（如 GEOIP,CN）与真正的服务签名。
// 纯国家代码不构成服务维度，不写进 SmartTarget。
var countryCodeRegex = regexp.MustCompile(`(?i)^[A-Z]{2}$`)

// smartRuleType 判断某条规则是否可作为 smart 组的"服务签名"。
func smartRuleType(rt C.RuleType) bool {
	return C.SmartRuleTypes[rt]
}

// ShouldStopRetry 判断该错误是否属于"重试也不会成功"的类型。
//
// smart 组在并行拨号时用它尽早放弃，避免在明显不可达的节点上反复消耗时间。
// 来自上游 mihomo。
func ShouldStopRetry(err error) bool {
	if errors.Is(err, resolver.ErrIPNotFound) {
		return true
	}
	if errors.Is(err, resolver.ErrIPVersion) {
		return true
	}
	if errors.Is(err, resolver.ErrIPv6Disabled) {
		return true
	}
	if errors.Is(err, loopback.ErrReject) {
		return true
	}
	return false
}
