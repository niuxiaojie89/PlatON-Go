package main

import (
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/p2p/discover"
)

const DefaultInitLevel = 1

type LevelNode struct {
	scans            []*discover.Node // 遍历节点列表
	actives          []*discover.Node // 活跃节点列表
	inactives        []*discover.Node // 不活跃节点列表
	currentScanIndex int              // 当前遍历到的节点index

	level int
	//lock  sync.RWMutex
}

func NewLevelNode(level int) *LevelNode {
	return &LevelNode{
		scans:            make([]*discover.Node, 0),
		actives:          make([]*discover.Node, 0),
		inactives:        make([]*discover.Node, 0),
		currentScanIndex: -1,
		level:            level,
	}
}

// 返回当前正在请求的节点
func (levelNode *LevelNode) currentScan() *discover.Node {
	//levelNode.lock.RLock()
	//defer levelNode.lock.RUnlock()
	index := levelNode.currentScanIndex
	if index == -1 || index >= len(levelNode.scans) {
		return nil
	}
	return levelNode.scans[index]
}

// 返回下一个将请求的节点
// 不存在可遍历的节点直接返回 false
// 存在可遍历节点返回节点并修改 currentScan
func (levelNode *LevelNode) nextScan() (bool, *discover.Node) {
	//levelNode.lock.Lock()
	//defer levelNode.lock.Unlock()
	nextIndex := levelNode.currentScanIndex + 1
	if nextIndex >= len(levelNode.scans) { // 最后一个元素
		return false, nil
	}
	levelNode.currentScanIndex++
	return true, levelNode.scans[nextIndex]
}

func (levelNode *LevelNode) addScan(n *discover.Node) error {
	//levelNode.lock.Lock()
	//defer levelNode.lock.Unlock()
	levelNode.scans = append(levelNode.scans, n)
	return nil
}

func (levelNode *LevelNode) addActive(n *discover.Node) error {
	//levelNode.lock.Lock()
	//defer levelNode.lock.Unlock()
	levelNode.actives = append(levelNode.actives, n)
	return nil
}

func (levelNode *LevelNode) printActive() string {
	resp := "["
	for i, n := range levelNode.actives {
		if i == len(levelNode.actives)-1 {
			resp += n.ID.TerminalString()
		} else {
			resp += n.ID.TerminalString() + " "
		}
	}
	resp += "]"
	return resp
}

func (levelNode *LevelNode) addInActive(n *discover.Node) error {
	//levelNode.lock.Lock()
	//defer levelNode.lock.Unlock()
	levelNode.inactives = append(levelNode.inactives, n)
	return nil
}

func (levelNode *LevelNode) printInActive() string {
	resp := "["
	for i, n := range levelNode.inactives {
		if i == len(levelNode.inactives)-1 {
			resp += n.ID.TerminalString()
		} else {
			resp += n.ID.TerminalString() + " "
		}
	}
	resp += "]"
	return resp
}

type LevelNodes struct {
	TargetID     discover.NodeID
	LevelMap     map[int]*LevelNode
	CurrentLevel int // 当前遍历到的深度
	AllNodes     map[discover.NodeID]struct{}

	scanDepth int
	//lock      sync.RWMutex
}

func NewLevelNodes(targetID discover.NodeID, scanDepth int) *LevelNodes {
	return &LevelNodes{
		TargetID: targetID,
		LevelMap: map[int]*LevelNode{
			DefaultInitLevel: NewLevelNode(DefaultInitLevel),
		},
		CurrentLevel: DefaultInitLevel,
		AllNodes:     make(map[discover.NodeID]struct{}),
		scanDepth:    scanDepth,
	}
}

func (levelNodes *LevelNodes) AddScan(level int, n *discover.Node, activeFlag bool) error {
	//levelNodes.lock.Lock()
	//defer levelNodes.lock.Unlock()

	if _, ok := levelNodes.AllNodes[n.ID]; ok {
		return fmt.Errorf("existing node, nodeID: %s", n.ID.TerminalString())
	}
	if _, ok := levelNodes.LevelMap[level]; !ok {
		levelNode := NewLevelNode(level) // 创建LevelNode对象
		levelNodes.LevelMap[level] = levelNode
	}
	levelNode, _ := levelNodes.LevelMap[level]
	levelNode.addScan(n)
	levelNodes.AllNodes[n.ID] = struct{}{}
	if activeFlag {
		levelNode.addActive(n)
	}
	return nil
}

// 返回下一个将请求的节点
// 当前 Level 不存在可请求节点，遍历下一个 Level map 并更新 CurrentLevel
func (levelNodes *LevelNodes) NextScan() (bool, *discover.Node, int) {
	//levelNodes.lock.Lock()
	//defer levelNodes.lock.Unlock()

	if ln, ok := levelNodes.LevelMap[levelNodes.CurrentLevel]; ok {
		if exists, node := ln.nextScan(); exists {
			return exists, node, levelNodes.CurrentLevel
		}
		if levelNodes.CurrentLevel+1 < levelNodes.scanDepth { // 递归深度不超过 scanDepth，即：默认第10层的节点不再继续 findnode
			if ln, ok := levelNodes.LevelMap[levelNodes.CurrentLevel+1]; ok {
				levelNodes.CurrentLevel++
				exists, node := ln.nextScan()
				return exists, node, levelNodes.CurrentLevel
			}
		}
	}
	return false, nil, 0
}

func (levelNodes *LevelNodes) AddActive(level int, n *discover.Node) error {
	//levelNodes.lock.Lock()
	//defer levelNodes.lock.Unlock()

	if _, ok := levelNodes.AllNodes[n.ID]; !ok {
		return fmt.Errorf("not existing node, nodeID: %s", n.ID.TerminalString())
	}

	ln, ok := levelNodes.LevelMap[level]
	if !ok {
		return fmt.Errorf("not existing level node, level: %d, level: %s", level, n.ID.TerminalString())
	}
	ln.addActive(n)
	return nil
}

func (levelNodes *LevelNodes) AddInActive(level int, n *discover.Node) error {
	//levelNodes.lock.Lock()
	//defer levelNodes.lock.Unlock()

	if _, ok := levelNodes.AllNodes[n.ID]; !ok {
		return fmt.Errorf("not existing node, nodeID: %s", n.ID.TerminalString())
	}

	ln, ok := levelNodes.LevelMap[level]
	if !ok {
		return fmt.Errorf("not existing level node, level: %d, level: %s", level, n.ID.TerminalString())
	}
	ln.addInActive(n)
	return nil
}

// 将当前待 scan 的节点全部装进 active 列表
// 一般用于总任务超时后，单个任务退出前的处理
func (levelNodes *LevelNodes) FillActive() error {
	//levelNodes.lock.Lock()
	//defer levelNodes.lock.Unlock()

	for {
		if ln, ok := levelNodes.LevelMap[levelNodes.CurrentLevel]; ok {
			if exists, node := ln.nextScan(); exists {
				ln.addActive(node)
			} else {
				levelNodes.CurrentLevel++
			}
		} else {
			break
		}
	}
	return nil
}

func (levelNodes *LevelNodes) Fstring() string {
	//levelNodes.lock.Lock()
	//defer levelNodes.lock.Unlock()

	resp := fmt.Sprintf("[")
	for level, ln := range levelNodes.LevelMap {
		resp += fmt.Sprintf("\nlevel: %d, actives count: %d, inactives count: %d", level, len(ln.actives), len(ln.inactives))
		resp += fmt.Sprintf("\nactives: %s", ln.printActive())
		resp += fmt.Sprintf("\ninactives: %s", ln.printInActive())
	}
	resp += fmt.Sprintf("\n]")
	return resp
}
