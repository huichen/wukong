持久存储
====

悟空引擎支持将搜索数据存入硬盘，并在启动时从硬盘恢复数据。使用持久存储只需启动EngineInitOptions中的三个选项：

```go
type EngineInitOptions struct {
  // 略过其他选项

  // 是否使用持久数据库，以及数据库文件保存的目录和裂分数目
  UsePersistentStorage bool
  PersistentStorageFolder string
  PersistentStorageShards int
```

当UsePersistentStorage为true时使用持久存储：

1. 在引擎启动时（engine.Init函数），引擎从PersistentStorageFolder指定的目录中读取
数据库保存的文档索引数据，重新计算索引表并给排序器注入排序数据。如果分词器的代码
或者词典有变化，这些变化会体现在新的索引表中。

2. 在调用engine.IndexDocument时，引擎将索引数据写入到PersistentStorageFolder指定
的目录中。

3. PersistentStorageShards定义了数据库裂分数目，默认为CPU数目。
