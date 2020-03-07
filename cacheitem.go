// 封装了对缓存条目的操作

package cache2go

import (
	// 同步相关
	"sync"
	// 时间相关
	"time"
)

// 缓存条目(key<->data)
type CacheItem struct {
	// 读写锁，匿名内嵌结构体
	// 保证CacheItem同步访问
	sync.RWMutex

	// key，可以是任意类型(空接口)
	key interface{}
	// data，可以是任意类型(空接口)
	data interface{}
	// 不被访问后的保活时间
	// 等于0说明永久保活
	lifeSpan time.Duration

	// 条目创建的时间
	createdOn time.Time
	// 最近一次访问条目的时间
	accessedOn time.Time
	// 条目被访问的次数
	accessCount int64

	// 条目被移除时的回调函数组
	// 元素是函数的切片
	aboutToExpire []func(key interface{}) 
}

// 创建条目
func NewCacheItem(key interface{}, lifeSpan time.Duration, data interface{}) *CacheItem {
	t := time.Now()
	return &CacheItem {
		key: key,
		data: data,
		lifeSpan: lifeSpan,
		createdOn: t,
		accessedOn: t,
		accessCount: 0,
		aboutToExpire: nil,
	}
}

// 重置被访问的时间，保活，避免被移除
func (item *CacheItem) KeepAlive() {
	item.Lock()
	defer item.Unlock()
	item.accessedOn = time.Now()
	item.accessCount++
}

// 返回不被访问后的保活时间
func (item *CacheItem) LifeSpan() time.Duration {
	// 不需要加锁，因为创建后就没有情况会修改此值
	return item.lifeSpan
}

// 返回最近一次访问条目的时间
func (item *CacheItem) AccessedOn() time.Time {
	item.RLock()
	defer item.RUnlock()
	return item.accessedOn
}

// 返回条目创建的时间
func (item *CacheItem) CreatedOn() time.Time {
	// 不需要加锁，因为创建后就没有情况会修改此值
	return item.createdOn
}

// 返回条目被访问的次数
func (item *CacheItem) accessCount() int64 {
	item.RLock()
	defer item.RUnlock()
	return item.accessCount
}

// 返回条目key
func (item *CacheItem) Key() interface{} {
	// 不需要加锁，因为创建后就没有情况会修改此值
	return item.key
}

// 返回条目data
func (item *CacheItem) Data() interface{} {
	// 不需要加锁，因为创建后就没有情况会修改此值
	return item.data
}

// 设置被移除时候的回调函数
func (item *CacheItem) SetAboutToExpireCallback() {
	if len(item.aboutToExpire) > 0 {
		item.RemoveAboutToExpireCallback()
	}
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = append(item.aboutToExpire, f)
}

// 添加被移除时候的回调函数
func (item *CacheItem) AddAboutToExpireCallback(f func(interface{})) {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = append(item.aboutToExpire, f)
}

// 删除被移除时候的回调函数
func (item *CacheItem) RemoveAboutToExpireCallback() {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = nil
}