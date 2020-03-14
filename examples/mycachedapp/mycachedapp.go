// cache其他接口的使用实例

package main

import (
	"log"
	"os"
	"fmt"
	"time"

	"../../../cache2go_annotated"
)

// value
type myStruct struct {
	text string
	moreData []byte
}

func main() {
	l := log.New(os.Stdout, "mycachedapp ", log.Ldate|log.Ltime)

	cache := cache2go.Cache("myCache")

	// 设置日志
	cache.SetLogger(l)

	// 空切片
	val := myStruct{"This is a test!", []byte{}}
	cache.Add("someKey", 5 * time.Second, &val)

	res, err := cache.Value("someKey")
	if err == nil {
		fmt.Println("Found value in cache:", res.Data().(*myStruct).text)
	} else {
		fmt.Println("Error retrieving value from cache:", err)
	}

	time.Sleep(6 * time.Second)
	res, err = cache.Value("someKey")
	if err != nil {
		fmt.Println("Item is not cached (anymore).")
	}

	cache.Add("someKey", 0, &val)

	// 设置删除缓存条目时触发的回调函数，会删除以前的回调函数
	cache.SetAboutToDeleteItemCallback(func(e *cache2go.CacheItem) {
		fmt.Println("Deleting:", e.Key(), e.Data().(*myStruct).text, e.CreatedOn())
	})

	cache.Delete("someKey")

	cache.Flush()
}