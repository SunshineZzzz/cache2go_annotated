// 封装了对缓存表项的操作

/*
 * Simple caching library with expiration capabilities
 *     Copyright (c) 2013-2017, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE.txt
 */

package cache2go

import (
	"log"
	"sort"
	"sync"
	"time"
)

// 缓存中的一个表项
type CacheTable struct {
	// 匿名对象，读写锁
	sync.RWMutex

	// 表项名称
	name string
	// 该表项存储的所有条目
	items map[interface{}]*CacheItem

	// 负责触发清除操作的计时器
	cleanupTimer *time.Timer
	// 触发清除操作的时间间隔
	cleanupInterval time.Duration

	// 该表项所使用的日志
	logger *log.Logger

	// 加载一个不存在的key时触发的回调函数，args可变长函数参数
	// 返回非nil，则加入到表中
	loadData func(key interface{}, args ...interface{}) *CacheItem
	// 添加缓存条目时触发的回调函数组
	addedItem []func(item *CacheItem)
	// 删除缓存条目时触发的回调函数组
	aboutToDeleteItem []func(item *CacheItem)
}

// 返回该表项拥有的条目个数
func (table *CacheTable) Count() int {
	table.RLock()
	defer table.RUnlock()
	return len(table.items)
}

// 遍历缓存条目，触发回调函数
func (table *CacheTable) Foreach(trans func(key interface{}, item *CacheItem)) {
	table.RLock()
	defer table.RUnlock()

	for k, v := range table.items {
		trans(k, v)
	}
}

// 设置加载一个不存在的key时触发的回调函数
func (table *CacheTable) SetDataLoader(f func(interface{}, ...interface{}) *CacheItem) {
	table.Lock()
	defer table.Unlock()
	table.loadData = f
}

// 设置添加缓存条目时触发的回调函数，会删除以前的回调函数
func (table *CacheTable) SetAddedItemCallback(f func(*CacheItem)) {
	if len(table.addedItem) > 0 {
		table.RemoveAddedItemCallbacks()
	}
	table.Lock()
	defer table.Unlock()
	table.addedItem = append(table.addedItem, f)
}

// 添加缓存条目时触发的回调函数
func (table *CacheTable) AddAddedItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.addedItem = append(table.addedItem, f)
}

// 删除添加缓存条目时触发的回调函数
func (table *CacheTable) RemoveAddedItemCallbacks() {
	table.Lock()
	defer table.Unlock()
	table.addedItem = nil
}

// 设置删除缓存条目时触发的回调函数，会删除以前的回调函数
func (table *CacheTable) SetAboutToDeleteItemCallback(f func(*CacheItem)) {
	if len(table.aboutToDeleteItem) > 0 {
		table.RemoveAboutToDeleteItemCallback()
	}

	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = append(table.aboutToDeleteItem, f)
}

// 添加删除缓存条目时触发的回调函数
func (table *CacheTable) AddAboutToDeleteItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = append(table.aboutToDeleteItem, f)
}

// 删除缓存条目时触发的回调函数
func (table *CacheTable) RemoveAboutToDeleteItemCallback() {
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = nil
}

// 设置日志
func (table *CacheTable) SetLogger(logger *log.Logger) {
	table.Lock()
	defer table.Unlock()
	table.logger = logger 
}

// 过期检查，能自动调节间隔
// 自动调节到条目中最早过期的时间，方便到时删除该条目
func (table *CacheTable) expirationCheck() {
	table.Lock()

	// 计时器停止，后面调整间隔后启动
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}

	// 实际并不会使用这个时间
	// if table.cleanupInterval > 0 {
	// 	table.log("Expiration check triggered after", 
	// 		table.cleanupInterval, "for table", table.name)
	// } else {
	// 	table.log("Expiration check installed for table", table.name)
	// }

	// 每次会更新
	now := time.Now()
	// 最小时间间隔
	// 遍历每个条目 保活时间 与 最近访问时间 差的最小值
	// 目的是 timer 到时候后，删除该条目
	smallestDuration := 0 * time.Second
	// 遍历所有的items查找最近一个将要过期的时间间隔
	for key, item := range table.items {
		item.RLock()

		lifeSpan := item.lifeSpan
		accessedOn := item.accessedOn
		
		item.RUnlock()

		// 永久报活
		if lifeSpan == 0 {
			continue
		}

		// 已经过期了，删除
		if now.Sub(accessedOn) >= lifeSpan {
			table.Unlock()
			
			// 内部删除接口
			table.deleteInternal(key)

			table.Lock()
		} else {
			// 更新smallestDuration，获取最近一个将要过期的时间间隔
			if smallestDuration == 0 || lifeSpan - now.Sub(accessedOn) < smallestDuration {
				smallestDuration = lifeSpan - now.Sub(accessedOn)
			}
		}
	}

	// 设置cleanupInterval为最近将要过期的时间间隔
	table.cleanupInterval = smallestDuration
	if smallestDuration > 0 {
		// 重新启动下一次的过期检测
		table.cleanupTimer = time.AfterFunc(smallestDuration, func() {
			go table.expirationCheck()
		})
	}

	table.Unlock()
}

// 内部添加函数，代码重用
func (table *CacheTable) addInternal(item *CacheItem) {
	table.log("Adding item with key", item.key, 
		"and lifeSpan of", item.lifeSpan, 
		"to table", table.name)

	table.items[item.key] = item

	// 表 触发清除操作的时间间隔
	expDur := table.cleanupInterval
	// 表 增加条目的回调函数组
	addedItem := table.addedItem

	table.Unlock()

	// 存在回掉函数就call
	if addedItem != nil {
		for _, callback := range addedItem {
			callback(item)
		}
	}

	// 如果当前过期检测时间等于0 或者 
	// 当前添加条目的保活时间 比当前 最短的过期时间还早过期，
	// 则主动触发过期检测函数
	// 
	// cleanupInterval 默认是0，只要加入了带有保活时间的条目
	// 就会触发检测函数注册相关
	if item.lifeSpan > 0 && (expDur == 0 || item.lifeSpan < expDur) {
		table.expirationCheck()
	}
}

// 创建缓存条目并且加入到缓存表
// 存在相同条目被前后增加的情况，不会并发增加
func (table *CacheTable) Add(key interface{}, lifeSpan time.Duration, data interface{}) *CacheItem {
	// 创建条目
	item := NewCacheItem(key, lifeSpan, data)

	table.Lock()
	// 内部添加接口
	table.addInternal(item)

	return item
}

// 内部删除函数，代码重用
// 存在一个 删除表中条目或条目被删除的回调函数 被多次调用的
// 情况，但是不会多次删除同一条目
func (table *CacheTable) deleteInternal(key interface{}) (*CacheItem, error) {
	table.Lock()

	r, ok := table.items[key]
	if !ok { 
		return nil, ErrKeyNotFound
	}

	aboutToDeleteItem := table.aboutToDeleteItem

	table.Unlock()

	// 触发删条目的回调函数
	if aboutToDeleteItem != nil {
		for _, callback := range aboutToDeleteItem {
			callback(r)
		}
	}

	// 触发条目过期删除的回调函数
	r.RLock()
	defer r.RUnlock()
	if r.aboutToExpire != nil {
		for _, callback := range r.aboutToExpire {
			callback(key)
		}
	}

	table.Lock()

	table.log("Deleting item with key", key, 
		"created on", r.createdOn, "and hit", 
		r.accessCount, "times from table", table.name)
	delete(table.items, key)
	
	table.Unlock()
	
	return r, nil
}

// 从缓存表中删除缓存条目
func (table *CacheTable) Delete(key interface{}) (*CacheItem, error) {
	return table.deleteInternal(key)
}

// 是否存在某个key
func (table *CacheTable) Exists(key interface{}) bool {
	table.RLock()
	defer table.RUnlock()
	_, ok := table.items[key]

	return ok
}

// 不存key就添加
func (table *CacheTable) NotFoundAdd(key interface{}, lifeSpan time.Duration, data interface{}) bool {
	table.Lock()

	if _, ok := table.items[key]; ok {
		table.Unlock()
		return false
	}

	// table.Unlock()，这里不应该解锁，
	// 增加完成后才可以解锁

	item := NewCacheItem(key, lifeSpan, data)
	table.addInternal(item)

	return true
}

// 获取value, 会通过KeepAlive更新访问时间和访问次数
func (table *CacheTable) Value(key interface{}, args ...interface{}) (*CacheItem, error) {
	table.RLock()

	r, ok := table.items[key]
	loadData := table.loadData

	table.RUnlock()

	if ok {
		r.KeepAlive()
		return r, nil
	}

	// 条目不存在
	if loadData != nil {
		// 打散slice
		item := loadData(key, args...)
		if item != nil {
			// 如果该key不存在，并发会造成相同的key多次被加入表中，
			// 从而造成key对应的内容被覆盖，应该调用
			// table.NotFoundAdd(key, item.lifeSpan, item.data)
			table.Add(key, item.lifeSpan, item.data)
			return item, nil
		}

		return nil, ErrKeyNotFoundOrLoadable
	}

	return nil, ErrKeyNotFound
}

// 清除所有的缓存条目，不会调用 缓存表的aboutToDeleteItem 和 缓存条目的aboutToExpire 
func (table *CacheTable) Flush() {
	table.Lock()
	defer table.Unlock()

	table.log("Flushing table", table.name)

	table.items = make(map[interface{}]*CacheItem)
	table.cleanupInterval = 0
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
}

// 内部打印日志
func (table *CacheTable) log(v ...interface{}) {
	if table.logger == nil {
		return
	}

	table.logger.Println(v...)
}

// 缓存条目<->访问次数对
type CacheItemPair struct {
	Key interface{}
	AccessCount int64
}

// 缓存条目<->访问次数对的切片
type CacheItemPairList []CacheItemPair

// qsort需要的一些函数， 根据访问次数排序
func (p CacheItemPairList) Swap(i, j int) { 
	p[i], p[j] = p[j], p[i] 
}
func (p CacheItemPairList) Len() int { 
	return len(p) 
}
func (p CacheItemPairList) Less(i, j int) bool { 
	return p[i].AccessCount > p[j].AccessCount 
}

// 获取访问最多的几个CacheItem，最多返回count个条目
func (table *CacheTable) MostAccessed(count int64) []*CacheItem {
	table.RLock()
	defer table.RLock()

	p := make(CacheItemPairList, len(table.items))
	i := 0
	for k, v := range table.items {
		p[i] = CacheItemPair{k, v.accessCount}
		i++
	}
	sort.Sort(p)

	var r []*CacheItem
	c := int64(0)
	for _, v := range p {
		if c >= count {
			break
		}

		item, ok := table.items[v.Key]
		if ok {
			r = append(r, item)
		}
		c++
	}

	return r
}