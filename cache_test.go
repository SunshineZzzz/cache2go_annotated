// 单元测试

/*
 * Simple caching library with expiration capabilities
 *     Copyright (c) 2013-2017, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE.txt
 */

package cache2go

import (
	"bytes"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 通用的key和value
var (
	k = "testkey"
	v = "testvalue"
)

// 表增加测试
func TestCache(t *testing.T) {
	table := Cache("testCache")
	// 永久保活
	table.Add(k + "_1", 0 * time.Second, v)
	// 保活1秒
	table.Add(k + "_2", 1 * time.Second, v)

	// 条目获取检测
	p, err := table.Value(k + "_1")
	if err != nil || p == nil || p.Data().(string) != v {
		t.Error("Error retrieving non expiring data from cache", err)
	}
	p, err = table.Value(k + "_2")
	if err != nil || p == nil || p.Data().(string) != v {
		t.Error("Error retrieving data from cache", err)
	}

	// 完整性检测
	// 条目被访问次数
	if p.AccessCount() != 1 {
		t.Error("Error getting correct access count")
	}
	// 没有访问后的保活时间
	if p.LifeSpan() != 1 * time.Second {
		t.Error("Error getting correct life-span")
	}
	// 最近一次访问条目的保活时间的UNIX时间戳
	if p.AccessedOn().Unix() == 0 {
		t.Error("Error getting access time")
	}
	// 条目创建的时间的UNI时间戳
	if p.CreatedOn().Unix() == 0 {
		t.Error("Error getting creation time")
	}
}

// 保活时间检测
func TestCacheExpire(t *testing.T) {
	table := Cache("testCache")

	table.Add(k + "_1", 100 * time.Millisecond, v + "_1")
	table.Add(k + "_2", 125 * time.Millisecond, v + "_2")

	time.Sleep(75 * time.Millisecond)

	// 活着
	_, err := table.Value(k + "_1")
	if err != nil {
		t.Error("Error retrieving value from cache:", err)
	}

	time.Sleep(75 * time.Millisecond)

	// 活着
	_, err = table.Value(k + "_1")
	if err != nil {
		t.Error("Error retrieving value from cache:", err)
	}

	// 被删除了
	_, err = table.Value(k + "_2")
	if err == nil {
		t.Error("Found key which should have been expired by now")
	}
}

// 条目是否存在测试
func TestExists(t *testing.T) {
	table := Cache("testExists")
	table.Add(k, 0, v)
	if !table.Exists(k) {
		t.Error("Error verifying existing data in cache")
	}
}

// 表中不存在条目就添加测试
func TestNotFoundAdd(t *testing.T) {
	table := Cache("testNotFoundAdd")

	if !table.NotFoundAdd(k, 0, v) {
		t.Error("Error verifying NotFoundAdd, data not in cache")
	}

	if table.NotFoundAdd(k, 0, v) {
		t.Error("Error verifying NotFoundAdd data in cache")
	}
}

// 表中不存在条目就添加的并发测试
func TestNotFoundAddConcurrency(t *testing.T) {
	table := Cache("testNotFoundAdd")

	// 倒计时计数器
	var finish sync.WaitGroup
	// 真实增加到表中的数量，正确是 100
	var added int32
	// 没有增加成功的，正确是 900
	var idle int32

	fn := func(id int) {
		for i := 0; i < 100; i++ {
			if table.NotFoundAdd(i, 0, i + id) {
				atomic.AddInt32(&added, 1)
			} else {
				atomic.AddInt32(&idle, 1)
			}
			// 目的应该让出CPU
			time.Sleep(0)
		}
		finish.Done()
	}

	finish.Add(10)
	
	// 0 ~ 99
	go fn(0x0000)
	// 4352 ~ 4451
	go fn(0x1100)
	// 8704 ~ 8803
	go fn(0x2200)
	// 13056 ~ 13155
	go fn(0x3300)
	// 17408 ~ 17507
	go fn(0x4400)
	// 21760 ~ 21859
	go fn(0x5500)
	// 26112 ~ 26211
	go fn(0x6600)
	// 30464 ~ 30563
	go fn(0x7700)
	// 34816 ~ 34915
	go fn(0x8800)
	// 39168 ~ 39267
	go fn(0x9900)

	finish.Wait()

	t.Log(added, idle)

	table.Foreach(func(key interface{}, item *CacheItem){
		v, _ := item.Data().(int)
		k, _ := key.(int)
		t.Logf("%02x %04x\n", k, v)
	})
}

// 检测条目的保活相关
func TestCacheKeepAlive(t *testing.T) {
	table := Cache("testKeepAlive")
	p := table.Add(k, 100 * time.Millisecond, v)

	time.Sleep(50 * time.Millisecond)
	p.KeepAlive()

	time.Sleep(75 * time.Millisecond)
	if !table.Exists(k) {
		t.Error("Error keeping item alive")
	}

	time.Sleep(75 * time.Millisecond)
	if table.Exists(k) {
		t.Error("Error expiring item after keeping it alive")
	}
}

// 测试删除条目接口
func TestDelete(t *testing.T) {
	table := Cache("testDelete")
	table.Add(k, 0, v)

	p, err := table.Value(k)
	if err != nil || p == nil || p.Data().(string) != v {
		t.Error("Error retrieving data from cache", err)
	}

	table.Delete(k)

	p, err = table.Value(k)
	if err == nil || p != nil {
		t.Error("Error deleting data")
	}

	_, err = table.Delete(k)
	if err == nil {
		t.Error("Expected error deleting item")
	}
}

// 测试flush接口
func TestFlush(t *testing.T) {
	table := Cache("testFlush")
	table.Add(k, 10 * time.Second, v)
	table.Flush()


	p, err := table.Value(k)
	if err == nil || p != nil {
		t.Error("Error flushing table")
	}

	if table.Count() != 0 {
		t.Error("Error verifying count of flushed table")
	}
}

// 测试Count接口
func TestCount(t *testing.T) {
	table := Cache("testCount")
	count := 100000
	for i := 0; i < count; i++ {
		key := k + strconv.Itoa(i)
		table.Add(key, 10 * time.Second, v)
	}

	for i := 0; i < count; i++ {
		key := k + strconv.Itoa(i)
		p, err := table.Value(key)
		if err != nil || p == nil || p.Data().(string) != v {
			t.Error("Error retrieving data")
		}
	}

	if table.Count() != count {
		t.Error("Data count mismatch")
	}
}

// 测试DataLoader接口
func TestDataLoader(t *testing.T) {
	table := Cache("testDataLoader")
	table.SetDataLoader(func(key interface{}, args ...interface{}) *CacheItem {
		var item *CacheItem
		if key.(string) != "nil" {
			val := k + key.(string)
			i := NewCacheItem(key, 500*time.Millisecond, val)
			item = i
		}

		return item	
	})

	_, err := table.Value("nil")
	if err == nil || table.Exists("nil") {
		t.Error("Error validating data loader for nil values")
	}

	for i := 0; i < 10; i++ {
		key := k + strconv.Itoa(i)
		vp := k + key
		p, err := table.Value(key)
		if err != nil || p == nil || p.Data().(string) != vp {
			t.Error("Error validating data loader")
		}
	}
}

// 测试MostAccessed接口
func TestAccessCount(t *testing.T) {
	count := 100
	table := Cache("testAccessCount")
	for i := 0; i < count; i++ {
		table.Add(i, 10 * time.Second, v)
	}

	// 0 - count, 1 - count - 1, ..., n - count - n
	for i := 0; i < count; i++ {
		for j := 0; j < i; j++ {
			table.Value(i)
		}
	}

	// 前100个访问最多的
	ma := table.MostAccessed(int64(count))
	for i, item := range ma {
		if item.Key() != count - 1 - i {
			t.Error("Most accessed items seem to be sorted incorrectly")
		}
	}

	ma = table.MostAccessed(int64(count - 1))
	if len(ma) != count - 1 {
		t.Error("MostAccessed returns incorrect amount of items")
	}
}

// 测试回调函数
func TestCallbacks(t *testing.T) {
	var m sync.Mutex
	addedKey := ""
	removedKey := ""
	calledAddedItem := false
	calledRemoveItem := false
	expired := false
	calledExpired := false

	table := Cache("testCallbacks")
	
	// 设置添加缓存条目时触发的回调函数，会删除以前的回调函数
	table.SetAddedItemCallback(func(item *CacheItem) {
		m.Lock()
		addedKey = item.Key().(string)
		m.Unlock()
	})
	table.SetAddedItemCallback(func(item *CacheItem) {
		m.Lock()
		calledAddedItem = true
		m.Unlock()
	})

	// 设置删除缓存条目时触发的回调函数，会删除以前的回调函数
	table.SetAboutToDeleteItemCallback(func(item *CacheItem) {
		m.Lock()
		removedKey = item.Key().(string)
		m.Unlock()
	})
	table.SetAboutToDeleteItemCallback(func(item *CacheItem) {
		m.Lock()
		calledRemoveItem = true
		m.Unlock()
	})

	i := table.Add(k, 500*time.Millisecond, v)
	// 设置条目被移除时候的回调函数，会删除以前的回调函数
	i.SetAboutToExpireCallback(func(key interface{}) {
		m.Lock()
		expired = true
		m.Unlock()
	})
	i.SetAboutToExpireCallback(func(key interface{}) {
		m.Lock()
		calledExpired = true
		m.Unlock()
	})

	time.Sleep(250 * time.Millisecond)

	m.Lock()
	if addedKey == k && !calledAddedItem {
		t.Error("AddedItem callback not working")
	}
	m.Unlock()

	time.Sleep(500 * time.Millisecond)
	m.Lock()
	if removedKey == k && !calledRemoveItem {
		t.Error("AboutToDeleteItem callback not working:" + k + "_" + removedKey)
	}
	if expired && !calledExpired {
		t.Error("AboutToExpire callback not working")
	}
	m.Unlock()
}

// 测试回调函数队列
func TestCallbackQueue(t *testing.T) {
	var m sync.Mutex
	addedKey := ""
	addedkeyCallback2 := ""
	secondCallbackResult := "second"
	removedKey := ""
	removedKeyCallback := ""
	expired := false
	calledExpired := false

	table := Cache("testCallbacks")

	// 添加缓存条目时触发的回调函数
	table.AddAddedItemCallback(func(item *CacheItem) {
		m.Lock()
		addedKey = item.Key().(string)
		m.Unlock()
	})
	table.AddAddedItemCallback(func(item *CacheItem) {
		m.Lock()
		addedkeyCallback2 = secondCallbackResult
		m.Unlock()
	})

	// 添加删除缓存条目时触发的回调函数
	table.AddAboutToDeleteItemCallback(func(item *CacheItem) {
		m.Lock()
		removedKey = item.Key().(string)
		m.Unlock()
	})
	table.AddAboutToDeleteItemCallback(func(item *CacheItem) {
		m.Lock()
		removedKeyCallback = secondCallbackResult
		m.Unlock()
	})

	i := table.Add(k, 500*time.Millisecond, v)
	// 添加被移除时候的回调函数
	i.AddAboutToExpireCallback(func(key interface{}) {
		m.Lock()
		expired = true
		m.Unlock()
	})
	i.AddAboutToExpireCallback(func(key interface{}) {
		m.Lock()
		calledExpired = true
		m.Unlock()
	})

	time.Sleep(250 * time.Millisecond)
	
	m.Lock()
	if addedKey != k && addedkeyCallback2 != secondCallbackResult {
		t.Error("AddedItem callback queue not working")
	}
	m.Unlock()

	time.Sleep(500 * time.Millisecond)

	m.Lock()
	if removedKey != k && removedKeyCallback != secondCallbackResult {
		t.Error("Item removed callback queue not working")
	}  
	m.Unlock()

	m.Lock()
	if !expired || !calledExpired {
		t.Error("Item expired callback queue not working")
	}
	m.Unlock()
	
	// 删除添加缓存条目时触发的回调函数
	table.RemoveAddedItemCallbacks()
	// 删除缓存条目时触发的回调函数
	table.RemoveAboutToDeleteItemCallback()

	secondItemKey := "itemKey02"
	expired = false
	i = table.Add(secondItemKey, 500*time.Millisecond, v)
	// 设置条目被移除时候的回调函数，会删除以前的回调函数
	i.SetAboutToExpireCallback(func(key interface{}) {
		m.Lock()
		expired = true
		m.Unlock()
	})
	// 删除被移除时候的回调函数
	i.RemoveAboutToExpireCallback()

	time.Sleep(250 * time.Millisecond)

	m.Lock()
	if addedKey == secondItemKey {
		t.Error("AddedItemCallbacks were not removed")
	}
	m.Unlock()

	time.Sleep(500 * time.Millisecond)

	m.Lock()
	if removedKey == secondItemKey {
		t.Error("AboutToDeleteItem not removed")
	}

	if !expired && !calledExpired {
		t.Error("AboutToExpire callback not working")
	}
	m.Unlock()
}

// 测试设置日志接口
func TestLogger(t *testing.T) {
	out := new(bytes.Buffer)
	l := log.New(out, "cache2go ", log.Ldate|log.Ltime)

	table := Cache("testLogger")
	table.SetLogger(l)

	table.Add(k, 0, v)

	time.Sleep(100 * time.Millisecond)

	if out.Len() == 0 {
		t.Error("Logger is empty")
	}
}
