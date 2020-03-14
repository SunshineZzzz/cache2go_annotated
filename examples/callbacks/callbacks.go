// callback实例

package main

import (
	"log"
	"os"
	"fmt"
	"time"

	"../../../cache2go_annotated"
)

func main() {
	l := log.New(os.Stdout, "callbacks ", log.Ldate|log.Ltime)

	cache := cache2go.Cache("myCache")

	// 设置日志
	cache.SetLogger(l)

	// 设置添加缓存条目时触发的回调函数，会删除以前的回调函数
	cache.SetAddedItemCallback(func(entry *cache2go.CacheItem){
		fmt.Println("Added Callback 1:", entry.Key(), entry.Data(), entry.CreatedOn())
	})
	// 添加缓存条目时触发的回调函数
	cache.AddAddedItemCallback(func(entry *cache2go.CacheItem){
		fmt.Println("Added Callback 2:", entry.Key(), entry.Data(), entry.CreatedOn())
	})
	// 设置删除缓存条目时触发的回调函数，会删除以前的回调函数
	cache.SetAboutToDeleteItemCallback(func(entry *cache2go.CacheItem) {
		fmt.Println("Deleting:", entry.Key(), entry.Data(), entry.CreatedOn())
	})

	cache.Add("someKey", 0, "This is a test!")

	res, err := cache.Value("someKey")
	if err == nil {
		fmt.Println("Found value in cache:", res.Data())
	} else {
		fmt.Println("Error retrieving value from cache:", err)
	}

	cache.Delete("someKey")

	// 删除添加缓存条目时触发的回调函数
	cache.RemoveAddedItemCallbacks()

	res = cache.Add("anotherKey", 3*time.Second, "This is another test")

	// 设置条目被移除时候的回调函数，会删除以前的回调函数
	res.SetAboutToExpireCallback(func(key interface{}) {
		fmt.Println("About to expire:", key.(string))
	})

	time.Sleep(5 * time.Second)
}