[cache2go代码的注释](https://github.com/muesli/cache2go)

#### 如何安装
```
go get github.com/SunshineZzzz/cache2go_annotated
cd $GOPATH/src/github.com/SunshineZzzz/cache2go_annotated
go get -u -v
go build
go test -v -run=.
go test -v -run=none -bench=.
```

#### 目录结构
```
.
├── benchmark_test.go 		基准测试
├── cache.go 				封装了对缓存的操作	
├── cacheitem.go 			封装了对缓存条目的操作
├── cachetable.go 			封装了对缓存表项的操作
├── cache_test.go 			单元测试
├── errors.go 				封装了对错误的描述
├── examples
│   ├── callbacks
│   │   └── callbacks.go 	callback使用案例
│   ├── dataloader
│   │   └── dataloader.go 	dataload使用案例
│   └── mycachedapp
│       └── mycachedapp.go 	其他常用接口使用案例
└── README.md
```

