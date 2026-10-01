// Package lightgbm 是 SLTE 对上游 mihomo-smart 中 LightGBM 预测引擎的替代实现。
//
// ## 为什么是空实现
//
// 上游该包会在智能组启用 `uselightgbm` 时，从**第三方 GitHub Release** 下载一个模型文件
// （硬编码 `https://github.com/vernesong/mihomo/releases/...`），并依赖第三方 Go 模块
// `github.com/vernesong/leaves` 做推理。对本项目而言这带来三个不可接受的问题：
//
//  1. 内核会在运行期主动外联到我们无法控制的地址（客户端的隐私与供应链风险）；
//  2. 该地址一旦失效，行为退化路径不可控；
//  3. 为一个可选增强引入一个低星第三方依赖。
//
// 因此 SLTE 的 smart 内核**只使用统计权重路径**（`smart.CalculateWeight`：连接成功率、
// 延迟、抖动、丢包等），这正是产品所需的"减少低延迟但实际连接异常"的能力。
// LightGBM 预测属于可选增强，暂无计划引入；将来若要启用，必须先自持模型分发。
//
// ## 兼容性
//
// 本包实现了上游同名包被 `adapter/outboundgroup/smart.go` 引用的**全部导出符号**，
// 因此那一侧代码与上游逐字节一致，便于后续跟进上游改动。
// `CreateModelInputFromStatsRecord` 是纯数据映射（不含任何模型逻辑），按上游原样保留，
// 以保证统计权重路径的输入结构完全一致。
package lightgbm

import (
	smart "github.com/metacubex/mihomo/component/smart"
	C "github.com/metacubex/mihomo/constant"
)

// DataCollector 是上游用于采集训练样本的组件。SLTE 不训练模型，因此是空壳。
type DataCollector struct{}

// GetCollector 返回空采集器；调用方对它 AddSample 不会产生任何副作用。
func GetCollector() *DataCollector {
	return &DataCollector{}
}

// AddSample 是空操作：不落盘、不上报。
func (c *DataCollector) AddSample(_ *smart.ModelInput, _ *C.Metadata, _ float64, _ string) {
}

// CloseAllCollectors 是空操作。
func CloseAllCollectors() {
}

// WeightModel 是上游的模型句柄。SLTE 不加载模型。
type WeightModel struct{}

// GetModel 恒返回 nil。
//
// smart.go 的调用点是 `if s.useLightGBM && s.weightModel != nil`，返回 nil 即稳定地走
// 统计权重分支；即使订阅内容把 `uselightgbm` 打开，也不会触发任何模型加载或下载。
func GetModel() *WeightModel {
	return nil
}

// PredictWeight 在 SLTE 的实现中永远不会被调用（GetModel 恒为 nil），
// 保留是为了满足接口形状；行为上与统计权重路径一致。
func (m *WeightModel) PredictWeight(input *smart.ModelInput, priorityFactor float64) (float64, bool) {
	return smart.CalculateWeight(input, priorityFactor)
}

// CreateModelInputFromStatsRecord 按上游原样保留：把统计快照映射为权重计算输入。
// 这是纯数据映射，不含模型逻辑。
func CreateModelInputFromStatsRecord(atomicRecord *smart.AtomicStatsRecord, metadata *C.Metadata, uploadTotal, downloadTotal, maxUploadRate, maxDownloadRate, connectionDuration float64, wildcardTarget string, lossRate, cumulLossRate float64) *smart.ModelInput {
	input := &smart.ModelInput{
		Success:                   atomicRecord.Get("success").(int64),
		Failure:                   atomicRecord.Get("failure").(int64),
		ConnectTime:               atomicRecord.Get("connectTime").(int64),
		Latency:                   atomicRecord.Get("latency").(int64),
		UploadTotal:               uploadTotal,
		HistoryUploadTotal:        atomicRecord.Get("uploadTotal").(float64),
		MaxuploadRate:             maxUploadRate,
		HistoryMaxUploadRate:      atomicRecord.Get("maxUploadRate").(float64),
		DownloadTotal:             downloadTotal,
		HistoryDownloadTotal:      atomicRecord.Get("downloadTotal").(float64),
		MaxdownloadRate:           maxDownloadRate,
		HistoryMaxDownloadRate:    atomicRecord.Get("maxDownloadRate").(float64),
		HistoryConnectionDuration: atomicRecord.Get("duration").(float64),
		ConnectionDuration:        connectionDuration,
		LastUsed:                  atomicRecord.Get("lastUsed").(int64),
		IsUDP:                     metadata.NetWork == C.UDP,
		IsTCP:                     metadata.NetWork == C.TCP,
		LossRate:                  lossRate,
		CumulLossRate:             cumulLossRate,
	}

	if metadata.DstIPASN == "unknown" {
		input.DestIPASN = ""
	} else {
		input.DestIPASN = metadata.DstIPASN
	}

	input.Host = wildcardTarget
	if metadata.DstIP.IsValid() {
		input.DestIP = metadata.DstIP.String()
	}

	input.DestPort = metadata.DstPort
	input.DestGeoIP = metadata.DstGeoIP

	return input
}
