package constant

// SmartRuleTypes 是会被 smart 组当作"服务签名"使用的规则类型。
//
// 命中这些规则时，隧道会把 `类型 [载荷]`（例如 `DOMAIN-SUFFIX [netflix.com]`）写进
// Metadata.SmartTarget，smart 组据此按服务维度分别记录与比较节点表现。
// 来自上游 mihomo（smart 组依赖）。
var SmartRuleTypes = map[RuleType]bool{
	Domain:         true,
	DomainSuffix:   true,
	DomainKeyword:  true,
	DomainRegex:    true,
	DomainWildcard: true,
	GEOSITE:        true,
	GEOIP:          true,
	IPASN:          true,
	IPCIDR:         true,
	IPSuffix:       true,
	RuleSet:        true,
	SubRules:       true,
	AND:            true,
	OR:             true,
}
