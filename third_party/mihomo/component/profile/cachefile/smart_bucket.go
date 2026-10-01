package cachefile

// bucketSmartStats 是 smart 组统计数据的存储桶。
//
// 上游把这一条写在 cache.go 的桶常量列表里；SLTE 单独成文件，避免为了一个常量
// 改动上游文件（也便于后续跟进上游）。
var bucketSmartStats = []byte("smart_stats")
