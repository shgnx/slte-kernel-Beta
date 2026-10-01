//go:build cmfa_smart

package outboundgroup

import (
	"github.com/metacubex/mihomo/common/structure"
	C "github.com/metacubex/mihomo/constant"
	P "github.com/metacubex/mihomo/constant/provider"
)

// newSmartProxyGroup 构建智能策略组（智能内核构建）。
func newSmartProxyGroup(groupOption GroupCommonOption, emptyFallback C.Proxy, providers []P.ProxyProvider, config map[string]any, decoder *structure.Decoder) (ProxyGroup, error) {
	opt := SmartOption{}
	if err := decoder.Decode(config, &opt); err != nil {
		return nil, err
	}

	return NewSmart(groupOption, opt, emptyFallback, providers)
}
