## nscan

运行 nscan 尽可能多地扫描出整个网络的节点，并输出节点 ID 和 IP。

## 说明

参考[文档](./节点扫描工具.md)。

## 编译

进入 `cmd/nscan` 目录，执行编译命令：

```
go build ./*
```
在 `cmd/nscan` 目录输出 `nscan` 可执行文件。

## 使用

执行 `./nscan -h` 查看启动选项：

### Connect to the PlatON network

| 选项 | 描述 |
| :------------ | :------------ |
| --chainID | 链ID，默认主网100 |
| --scanTime | 整个任务的扫描时间，超过扫描时间，终止扫描并输出当前结果。单位：小时，默认2小时 |
| --scanDepth | 扫描深度，默认10层 |
| --targetsNumber | targetID 个数，默认32 |
| --bootNodes | 种子节点列表，多个用逗号分隔。需和chainID匹配上 |
| --nodeData | 扫描结果输出文件，可以指定绝对路径文件也可以只指定文件名，默认创建在当前执行目录下 |