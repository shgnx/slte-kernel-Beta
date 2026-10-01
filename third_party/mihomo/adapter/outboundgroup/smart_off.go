//go:build !cmfa_smart

package outboundgroup

import (
	"fmt"

	"github.com/metacubex/mihomo/common/structure"
	C "github.com/metacubex/mihomo/constant"
	P "github.com/metacubex/mihomo/constant/provider"
)

// newSmartProxyGroup 在原版内核里明确拒绝：智能组是独立内核才有的能力，
// 原版内核必须保持"保持原有使用方式与兼容性"的定位。
//
// 走显式报错而不是静默降级：订阅里出现 type: smart 时配置加载会失败，
// 用户能立刻看出"这份订阅需要智能内核"，而不是拿到一个行为不明的组。
func newSmartProxyGroup(GroupCommonOption, C.Proxy, []P.ProxyProvider, map[string]any, *structure.Decoder) (ProxyGroup, error) {
	return nil, fmt.Errorf("%w: smart (需要智能内核)", errType)
}
