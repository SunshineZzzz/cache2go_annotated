// 封装了对错误的描述

package cache2go

import (
	"errors"
)

var (
	// key 不存在表中
	ErrKeyNotFound = errors.New("Key not found in cache")
	// key 不存在 或者 loadData 无法创建 条目
	ErrKeyNotFoundOrLoadable = errors.New("Key not found and could not be loaded into cache")
)