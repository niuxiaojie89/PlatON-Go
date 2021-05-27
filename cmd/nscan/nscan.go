package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/p2p/discover"
	"strings"
	"time"
)

const maxFindnodeFailures = 3

type ScanTaskResult struct {
	taskID     int
	targetID   discover.NodeID
	levelNodes *LevelNodes
}

var TaskTimeout = false

var TaskResult = make(map[int]ScanTaskResult)
var Actives = make(map[discover.NodeID]*discover.Node)
var InActives = make(map[discover.NodeID]*discover.Node)
var Revivals = make(map[discover.NodeID]*discover.Node)

func main() {
	// 扫描时间
	scanTime := flag.Int64("scanTime", 2, "The duration of the task, unit: hour, the default is 2 hour")
	// targetID 个数
	targetsNumber := flag.Int("targetsNumber", 32, "The number of targets used to discover neighbor nodes, the default is 32")
	// 扫描深度
	scanDepth := flag.Int("scanDepth", 10, "The depth of recursion for each targetID. Default 10")
	// bootnodes
	//bootNodes := *flag.String("bootNodes", "enode://6928c02520ff4e1784d49b4987eee9852dd7b1552f89836292de1002869a78697a9e697d96001e079b5c662135fd5234ae31b3a94e53283d56caf7652f6d6e90@seed1.a72b.alaya.network:16789,enode://3fdefc6e19e46cb05700b947ff8261087706697fe6054ddf925a261af2780084252bf7b2f6cf652e1fec4d64d3e9539ca9408a1b5ead5fe82b555d95cf143fb2@seed2.afc7.alaya.network:16789,enode://3fe92730eb9b53a2e58a9be11a1707c346432ee0c531c24a22d4bb2d0d4a9b4ef04e23988b4fa5d91a790c7f821e9983ae71b03903a3d75bfcce156b060cf99b@seed3.ccf5.alaya.network:16789,enode://1972a5a7d75010e199eac231ab513e564cad5f0e88331a53606b7d55220803c1816d3b0d06ca9b0e10389264f4fade77c46814dd44df502599d3f0a286160498@seed4.5e92.alaya.network:16789,enode://02dc695641f5cada2c685e3bf3dca0218a9dc7a5d5ce8165a2f5bee40d002d18ec6d899abaac1472d88b71e49691019766abd177b8d5d94f72f6f6dc842fded2@seed5.10a1.alaya.network:16789,enode://49648f184dab8acf0927238452e1f7e8f0e86135dcd148baa0a2d22cd931ed9770f726fccf13a3976bbdb738a880a074dc21d811037190267cd9c8c1378a6043@seed6.8d4b.alaya.network:16789,enode://5670e1b34fe39da46ebdd9c3377053c5214c7b3e6b371d31fcc381f788414a38d44cf844ad71305eb1d0d8afddee8eccafb4d30b33b54ca002db47c4864ba080@seed7.bdac.alaya.network:16789", "Comma separated enode URLs for P2P discovery bootstrap")
	//// chainID
	//chainID := *flag.Uint64("chainID", 201018, "The chainID. Default mainnet")
	bootNodes := flag.String("bootNodes", "enode://f7c33bd34b0e3c9a0317733ef3356409ff2eb009605cc357c213c367faf833ab48d557942731fd8dfdd39b92004e863c0fd8ebd01e79f69acd2b82f60ac63074@ms1.bfa6.platon.network:16789,enode://7729f2313908670523d7babf227e69e93150aac916f3f372a36ee7f204ed55737cb667fc55c34c40c5499734ed08cc7c57800d1ac8131c0cb855768801b898e9@ms2.6cc3.platon.network:16789,enode://4649f744f3e1d2400773fc48e057b96a8d4a10e00121f884f97b3182187ded0f89f5f4dbade55acaa4155e25c281f23a34587bad4fc3af2403eef9c130b57e5b@ms3.cd41.platon.network:16789,enode://f77401d3dda6d0c58310744e9349c16c056f94179a4d7bdc3470b6461d7f64370fa21ebf380ab10e25834291215715550c6845442a0df88ab1c42d161d367626@ms4.1fda.platon.network:16789,enode://8c71f4e1e795fc6e73144e4696a9fde3c3cdf6b99ab575357b77ab22542bc70c8f04e88f23eb6cdb225ad077aa67b71245da1ab1838bf8b362464ffd515ca3d6@ms5.ee7a.platon.network:16789,enode://6c9f8a51ff27bb0e062952be1f1e3943847eaed1f14d54783238678767dd134ddedfc82c0f52696cee971ee1aabec00523cd634e41c079ee95407aa4ecb92c7b@ms6.63a8.platon.network:16789,enode://0db310d4a6c429dcac973ff6433659ed710783872cf62bdcec09c76a8bb380d51f0c153401ceaa27e0b3de6f56fb939115c495862d15f6ccc8019702117be34d@ms7.66dc.platon.network:16789", "Comma separated enode URLs for P2P discovery bootstrap")
	// chainID
	chainID := flag.Uint64("chainID", 100, "The chainID. Default mainnet 100")
	// 节点扫描结果输出文件
	nodeData := flag.String("nodeData", "./nodes.txt", "Node scan result output file")
	flag.Parse()

	fmt.Println("Nscan params: ", "scanTime", *scanTime, "targetsNumber", *targetsNumber, "scanDepth", *scanDepth, "bootNodes", *bootNodes, "chainID", *chainID, "nodeData", *nodeData)

	// 解析 bootNodes
	if *bootNodes == "" {
		fmt.Println("Please enter bootNodes information")
		return
	}
	urls := strings.Split(*bootNodes, ",")
	bootstrapNodes, err := parseBootNodes(urls)
	if err != nil {
		fmt.Println("Failed to parse bootNodes", "error", err.Error())
		return
	}

	// 开启UDP服务
	ntab, err := NewUDPTable(*chainID)
	if err != nil {
		fmt.Println("Failed to start UDP service", "error", err.Error())
		return
	}
	defer ntab.Close()

	taskResult := make(chan ScanTaskResult, *targetsNumber)
	taskCount := 0
	for i := 0; i < *targetsNumber; i++ {
		// 随机生成一个targetID
		var target discover.NodeID
		rand.Read(target[:])
		// TODO
		//target := discover.MustHexID("0x57d9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439")
		// 向种子节点“递归”发起节点发现（每个 targetID 对应一个小任务）
		taskCount++
		go scanNodes(ntab, target, bootstrapNodes, *scanDepth, taskResult, taskCount)
	}

	//timer := time.NewTimer(30 * time.Second)
	scanTimeInterval := time.Duration(*scanTime) * time.Hour
	timer := time.NewTimer(scanTimeInterval)
	defer timer.Stop()

	startTime := time.Now()
running:
	for {
		select {
		case <-timer.C:
			fmt.Println("Task execution timeout, task termination...")
			TaskTimeout = true
		case result := <-taskResult:
			fmt.Println("ScanNodes task result: ", "taskID", result.taskID, "targetID", result.targetID.TerminalString(), "totalNodes", len(result.levelNodes.AllNodes), "levelNodes", result.levelNodes.Fstring())
			// 合并结果
			TaskResult[result.taskID] = result
			mergeResult(result.levelNodes)
			taskCount--
			if taskCount == 0 {
				break running
			}
		}
	}
	// 把 InActives 节点进行dial 判断是否在线
	revivals()
	duration := common.PrettyDuration(time.Since(startTime)).String()
	// 结果输出到文件
	OutputFile(*nodeData, duration)
	fmt.Println("Task completed: ", "duration", duration)
}

// 解析 bootNodes
func parseBootNodes(urls []string) ([]*discover.Node, error) {
	bootstrapNodes := make([]*discover.Node, 0, len(urls))
	for _, url := range urls {
		node, err := discover.ParseNode(url)
		if err != nil {
			return nil, fmt.Errorf("bootstrap URL invalid, enode: %s, error: %s", url, err.Error())

		}
		bootstrapNodes = append(bootstrapNodes, node)
	}
	return bootstrapNodes, nil
}

// 合并结果
func mergeResult(levelNodes *LevelNodes) {
	if levelNodes.LevelMap != nil {
		for _, levelNode := range levelNodes.LevelMap {
			for _, n := range levelNode.actives {
				if _, ok := Actives[n.ID]; !ok {
					Actives[n.ID] = n
				}
				if _, ok := InActives[n.ID]; ok {
					delete(InActives, n.ID) // 该 node 在别的任务中是不活跃的，但在本任务中是活跃的，那么整个结果看它也是活跃的
				}
			}
			for _, n := range levelNode.inactives {
				if _, ok1 := Actives[n.ID]; !ok1 {
					if _, ok2 := InActives[n.ID]; !ok2 {
						InActives[n.ID] = n
					}
				} else {
					delete(InActives, n.ID) // 该 node 在本任务中是不活跃的，但在别的任务中是活跃的，那么整个结果看它也是活跃的
				}
			}
		}
	}
}

// 把 InActives 节点进行dial 判断是否在线
func revivals() {
	tcpDialer := NewTCPDialer()
	for _, n := range InActives {
		fd, err := tcpDialer.Dial(n)
		if err == nil {
			fmt.Println("revival node", "nodeId", n.ID.TerminalString(), "nodeIP", n.Addr())
			Revivals[n.ID] = n
			fd.Close()
		}
	}
	// 从 InActives 中删除复活节点
	for id, _ := range Revivals {
		delete(InActives, id)
	}
}

// 扫描节点
func scanNodes(ntab *discover.Table, target discover.NodeID, bootstrapNodes []*discover.Node, scanDepth int, result chan ScanTaskResult, taskID int) {
	levelNodes := NewLevelNodes(target, scanDepth)
	for _, n := range bootstrapNodes {
		levelNodes.AddScan(DefaultInitLevel, n, false) // 种子节点放到第一层
	}
	// 开始扫描节点
	for {
		if TaskTimeout {
			// 总任务超时，退出子任务并返回当前扫描结果
			break
		}
		if exists, node, level := levelNodes.NextScan(); exists {
			if r, err := ntab.Findnode(node, target, maxFindnodeFailures); err == nil {
				// 将 node 标记为 active
				levelNodes.AddActive(level, node)
				newLevel := level + 1
				for _, n := range r {
					//fmt.Println("Findnode list", "taskID", taskID, "level", level, "fromID", node.ID.TerminalString(), "fromIP", node.Addr(), "nodeID", n.ID.TerminalString(), "nodeIP", n.Addr())
					if newLevel != scanDepth {
						// 将新发现的节点添加到 scan 列表
						levelNodes.AddScan(newLevel, n, false)
					} else {
						// 达到递归深度，不会基于最后一层节点继续做发现，该层节点默认进入 active 列表
						levelNodes.AddScan(newLevel, n, true)
					}
				}
			} else {
				// 将 node 标记为 inactive
				levelNodes.AddInActive(level, node)
			}
		} else {
			break
		}
	}
	levelNodes.FillActive()
	fmt.Println("Single task completed: ", "taskID", taskID, "targetID", target.TerminalString(), "scanDepth", scanDepth)
	result <- ScanTaskResult{
		taskID:     taskID,
		targetID:   target,
		levelNodes: levelNodes,
	}
}
