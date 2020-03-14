// dataloader实例

package main

import (
	"log"
	"os"
	"fmt"
	"strconv"

	"../../../cache2go_annotated"
)

func main() {
	l := log.New(os.Stdout, "dataloader ", log.Ldate|log.Ltime)

	cache := cache2go.Cache("myCache")

	// 设置日志
	cache.SetLogger(l)

	// 设置加载一个不存在的key时触发的回调函数
	// 该函数返回CacheItem就会加入表中
	cache.SetDataLoader(func(key interface{}, args ...interface{}) *cache2go.CacheItem {
		val := "This is a test with key " + key.(string)

		item := cache2go.NewCacheItem(key, 0, val)

		return item
	})

	for i := 0; i < 10; i++ {
		res, err := cache.Value("someKey_" + strconv.Itoa(i))
		if err == nil {
			fmt.Println("Found value in cache:", res.Data())
		} else {
			fmt.Println("Error retrieving value from cache:", err)
		}
	}
}