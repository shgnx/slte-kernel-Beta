package atomic

import (
	"encoding/json"
	"math"
	"sync/atomic"
)

// Update 按 f 原子更新当前值。
//
// 来自上游 mihomo（smart 组依赖）。f 在 CAS 失败时会被重新调用，
// 因此必须是无副作用的纯函数。
func (t *TypedValue[T]) Update(f func(old T) (new T)) {
	for {
		currentP := t.value.Load()
		var old T
		if currentP != nil {
			old = *currentP
		}

		newValue := f(old)
		if t.value.CompareAndSwap(currentP, &newValue) {
			return
		}
	}
}

// Float64 是 float64 的原子包装（以 IEEE-754 位模式存储）。
// 来自上游 mihomo，smart 组用它记录权重类统计量。
type Float64 struct {
	atomic.Uint64
}

func NewFloat64(val float64) (f Float64) {
	f.Store(val)
	return
}

func (f *Float64) Store(val float64) {
	f.Uint64.Store(math.Float64bits(val))
}

func (f *Float64) Load() float64 {
	return math.Float64frombits(f.Uint64.Load())
}

func (f *Float64) Add(delta float64) float64 {
	for {
		oldBits := f.Uint64.Load()
		old := math.Float64frombits(oldBits)
		newValue := old + delta
		newBits := math.Float64bits(newValue)
		if f.Uint64.CompareAndSwap(oldBits, newBits) {
			return newValue
		}
	}
}

func (f *Float64) Swap(newValue float64) float64 {
	oldBits := f.Uint64.Swap(math.Float64bits(newValue))
	return math.Float64frombits(oldBits)
}

func (f *Float64) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Load())
}

func (f *Float64) UnmarshalJSON(b []byte) error {
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	f.Store(v)
	return nil
}

func (f *Float64) MarshalYAML() (any, error) {
	return f.Load(), nil
}

func (f *Float64) UnmarshalYAML(unmarshal func(any) error) error {
	var v float64
	if err := unmarshal(&v); err != nil {
		return err
	}
	f.Store(v)
	return nil
}
