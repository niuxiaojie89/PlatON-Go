package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 结果输出到文件
func OutputFile(nodeData string, duration string) error {
	filePath := ResolvePath(nodeData)
	fmt.Println("outputFile", "filePath", filePath)
	if PathExists(filePath) {
		os.Remove(filePath)
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0775)
	if err != nil {
		return fmt.Errorf("open file error, filePath: %s, error: %s", filePath, err.Error())
	}
	// 关闭文件
	defer f.Close()

	f.WriteString("Task completed, duration: " + duration)
	for _, r := range TaskResult {
		resp := "\n任务，targetID: " + r.targetID.TerminalString() + "\n"
		level := DefaultInitLevel
		for {
			if level > len(r.levelNodes.LevelMap) {
				break
			}
			if ln, ok := r.levelNodes.LevelMap[level]; ok {
				resp += fmt.Sprintf("level: %d, actives count: %d, inactives count: %d \n", level, len(ln.actives), len(ln.inactives))
			}
			level++
		}
		f.WriteString(resp)
	}
	// Actives
	f.WriteString(fmt.Sprintf("\n活跃节点，总数: %d \n", len(Actives)))
	activesResp := ""
	for _, n := range Actives {
		activesResp += "[" + n.ID.TerminalString() + " " + n.Addr().String() + "]\n"
	}
	f.WriteString(activesResp)
	// InActives
	f.WriteString(fmt.Sprintf("\n不活跃节点，总数: %d \n", len(InActives)))
	inActivesResp := ""
	for _, n := range InActives {
		inActivesResp += "[" + n.ID.TerminalString() + " " + n.Addr().String() + "]\n"
	}
	f.WriteString(inActivesResp)

	// Revivals
	f.WriteString(fmt.Sprintf("\n复活节点，总数: %d \n", len(Revivals)))
	revivalsResp := ""
	for _, n := range Revivals {
		revivalsResp += "[" + n.ID.TerminalString() + " " + n.Addr().String() + "]\n"
	}
	f.WriteString(revivalsResp)
	return nil
}

// 获取当前路径
func GetCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}

// 返回完整文件路径
func ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	currentPath, _ := GetCurrentPath()
	return filepath.Join(currentPath, path)
}

// 判断文件是否存在
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
