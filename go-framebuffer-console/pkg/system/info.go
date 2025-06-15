// system包提供了系统信息获取功能
// 实现了Linux系统的硬件信息、网络状态、系统状态等信息的收集
// 专门针对CentOS 7.9环境进行了优化，确保信息获取的准确性和兼容性
package system

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SystemInfo 系统信息结构体
// 包含了系统运行状态、硬件配置、网络信息等核心数据
type SystemInfo struct {
	Uptime      string // 系统运行时间（格式化为天、小时、分钟）
	CPUModel    string // CPU型号名称
	CPUCores    int    // CPU核心数量
	MemoryUsage string // 内存使用情况（百分比和具体数值）
	DiskSize    string // 根分区磁盘大小
	DiskCount   int    // 磁盘设备数量
	CurrentTime string // 当前系统时间
	IPAddress   string // 主要网络接口的IP地址
}

func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	var err error
	info.Uptime, err = getUptime()
	if err != nil {
		info.Uptime = "未知"
	}

	info.CPUModel, info.CPUCores, err = getCPUInfo()
	if err != nil {
		info.CPUModel = "未知"
		info.CPUCores = runtime.NumCPU()
	}

	info.MemoryUsage, err = getMemoryUsage()
	if err != nil {
		info.MemoryUsage = "未知"
	}

	info.DiskSize, info.DiskCount, err = getDiskInfo()
	if err != nil {
		info.DiskSize = "未知"
		info.DiskCount = 0
	}

	info.CurrentTime = time.Now().Format("2006-01-02 15:04:05")

	info.IPAddress, err = getIPAddress()
	if err != nil {
		info.IPAddress = "未知"
	}

	return info, nil
}

func getUptime() (string, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "", fmt.Errorf("读取uptime文件失败: %v", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return "", fmt.Errorf("invalid uptime format")
	}

	uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "", fmt.Errorf("解析uptime数据失败: %v", err)
	}

	// 防止负数和过大的值
	if uptimeSeconds < 0 || uptimeSeconds > 365*24*3600*100 { // 限制100年
		return "", fmt.Errorf("不合理的uptime值: %f", uptimeSeconds)
	}

	days := int(uptimeSeconds) / 86400
	hours := (int(uptimeSeconds) % 86400) / 3600
	minutes := (int(uptimeSeconds) % 3600) / 60

	return fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes), nil
}

func getCPUInfo() (string, int, error) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "", 0, fmt.Errorf("读取CPU信息失败: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	cpuModel := ""
	cpuCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cpuModel = strings.TrimSpace(parts[1])
				// 防止过长的CPU名称
				if len(cpuModel) > 100 {
					cpuModel = cpuModel[:100] + "..."
				}
			}
		}
		if strings.HasPrefix(line, "processor") {
			cpuCount++
			// 防止过多的CPU核心数
			if cpuCount > 1024 {
				return "", 0, fmt.Errorf("CPU核心数过多: %d", cpuCount)
			}
		}
	}

	if cpuModel == "" {
		cpuModel = "未知处理器"
	}
	if cpuCount == 0 {
		cpuCount = runtime.NumCPU()
	}

	return cpuModel, cpuCount, nil
}

func getMemoryUsage() (string, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "", fmt.Errorf("读取内存信息失败: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	var memTotal, memAvailable int64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if val, parseErr := strconv.ParseInt(fields[1], 10, 64); parseErr == nil {
					memTotal = val
				}
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if val, parseErr := strconv.ParseInt(fields[1], 10, 64); parseErr == nil {
					memAvailable = val
				}
			}
		}
	}

	// 数据有效性检查
	if memTotal <= 0 || memTotal > 1024*1024*1024 { // 限制最大1TB
		return "未知", nil
	}
	if memAvailable < 0 || memAvailable > memTotal {
		memAvailable = 0
	}

	memUsed := memTotal - memAvailable
	usagePercent := float64(memUsed) / float64(memTotal) * 100

	return fmt.Sprintf("%.1f%% (已用: %s / 总计: %s)",
		usagePercent,
		formatBytes(memUsed*1024),
		formatBytes(memTotal*1024)), nil
}

func getDiskInfo() (string, int, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return "", 0, fmt.Errorf("获取磁盘信息失败: %v", err)
	}

	// 检查数据的合理性
	if stat.Blocks == 0 || stat.Bsize == 0 {
		return "未知", 1, nil
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	// 防止溢出
	if totalBytes > 1024*1024*1024*1024*1024 { // 限制最大1PB
		return "过大", 1, nil
	}

	diskSize := formatBytes(int64(totalBytes))
	diskCount := 1

	if data, err := os.ReadFile("/proc/mounts"); err == nil {
		lines := strings.Split(string(data), "\n")
		diskDevices := make(map[string]bool)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) >= 1 && strings.HasPrefix(fields[0], "/dev/") {
				diskDevices[fields[0]] = true
				// 防止过多设备
				if len(diskDevices) > 100 {
					break
				}
			}
		}
		diskCount = len(diskDevices)
		if diskCount == 0 {
			diskCount = 1
		}
	}

	return diskSize, diskCount, nil
}

func getIPAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
	}

	return "未获取到IP", nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func GetNetworkInterfaces() ([]NetworkInterface, error) {
	allInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var physicalInterfaces []NetworkInterface
	for _, iface := range allInterfaces {
		// 1. 排除Loopback接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 2. 通过sysfs检查是否为物理设备
		devicePath := fmt.Sprintf("/sys/class/net/%s/device", iface.Name)
		if _, err := os.Stat(devicePath); os.IsNotExist(err) {
			continue // 不存在device目录，判定为虚拟网卡
		}

		// 3. 获取IP地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var ipv4Addr string
		var ipv6s []string
		for _, addr := range addrs {
			var ip net.IP
			ipStr := addr.String()
			ip, _, err := net.ParseCIDR(ipStr)
			if err != nil {
				ip = net.ParseIP(strings.Split(ipStr, "/")[0])
			}

			if ip == nil {
				continue
			}

			if ip.To4() != nil {
				// 只取第一个非本地链路的IPv4地址
				if ipv4Addr == "" && !ip.IsLinkLocalUnicast() {
					ipv4Addr = ip.String()
				}
			} else {
				ipv6s = append(ipv6s, ip.String())
			}
		}

		status := "Down"
		if iface.Flags&net.FlagUp != 0 {
			status = "Up"
		}
		if iface.Flags&net.FlagRunning != 0 {
			status += ", Running"
		}

		physicalInterfaces = append(physicalInterfaces, NetworkInterface{
			Name:          iface.Name,
			Status:        status,
			MAC:           iface.HardwareAddr.String(),
			IPv4Address:   ipv4Addr,
			IPv6Addresses: ipv6s,
		})
	}

	return physicalInterfaces, nil
}

// NetworkInterface 包含了网络接口的详细信息
type NetworkInterface struct {
	Name          string
	Status        string
	MAC           string
	IPv4Address   string
	IPv6Addresses []string
}

// NetworkTestTarget 网络测试目标
type NetworkTestTarget struct {
	Name        string // 显示名称
	Host        string // 主机地址
	Description string // 描述
}

// NetworkTestResult 网络测试结果
type NetworkTestResult struct {
	Target       NetworkTestTarget
	Success      bool
	PacketsSent  int
	PacketsRecv  int
	PacketLoss   float64
	AvgLatency   string
	ErrorMsg     string
}

// NetworkTestProgress 网络测试进度回调
type NetworkTestProgress func(target string, current, total int, message string)

// TestNetworkConnectivity 简单网络测试（保持向后兼容）
func TestNetworkConnectivity() (bool, error) {
	return TestNetworkConnectivityWithTimeout(5 * time.Second)
}

func TestNetworkConnectivityWithTimeout(timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "3", "8.8.8.8")
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return false, fmt.Errorf("网络测试超时")
	}

	return err == nil, err
}

// TestAdvancedNetworkConnectivity 高级网络连通性测试
func TestAdvancedNetworkConnectivity(progressCallback NetworkTestProgress) ([]NetworkTestResult, error) {
	// 定义测试目标
	targets := []NetworkTestTarget{
		{Name: "字节跳动", Host: "bytedance.com", Description: "字节跳动官网"},
		{Name: "百度", Host: "baidu.com", Description: "百度首页"},
		{Name: "哔哩哔哩", Host: "bilibili.com", Description: "哔哩哔哩"},
		{Name: "腾讯", Host: "tencent.com", Description: "腾讯官网"},
		{Name: "阿里DNS", Host: "223.5.5.5", Description: "阿里云DNS服务器"},
	}

	results := make([]NetworkTestResult, len(targets))
	
	for i, target := range targets {
		if progressCallback != nil {
			progressCallback(target.Name, i+1, len(targets), fmt.Sprintf("正在测试 %s...", target.Description))
		}
		
		result := testSingleTarget(target)
		results[i] = result
		
		if progressCallback != nil {
			status := "成功"
			if !result.Success {
				status = "失败"
			}
			progressCallback(target.Name, i+1, len(targets), fmt.Sprintf("%s 测试%s", target.Description, status))
		}
	}
	
	return results, nil
}

// testSingleTarget 测试单个目标
func testSingleTarget(target NetworkTestTarget) NetworkTestResult {
	result := NetworkTestResult{
		Target:      target,
		PacketsSent: 4,
		PacketsRecv: 0,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	
	// 使用ping命令测试，发送4个包
	cmd := exec.CommandContext(ctx, "ping", "-c", "4", "-W", "3", target.Host)
	output, err := cmd.CombinedOutput()
	
	if ctx.Err() == context.DeadlineExceeded {
		result.ErrorMsg = "测试超时"
		result.PacketLoss = 100.0
		return result
	}
	
	if err != nil {
		result.ErrorMsg = fmt.Sprintf("ping失败: %v", err)
		result.PacketLoss = 100.0
		return result
	}
	
	// 解析ping输出结果
	outputStr := string(output)
	result.Success = true
	
	// 解析统计信息
	if strings.Contains(outputStr, "packets transmitted") {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			
			// 解析包统计: "4 packets transmitted, 4 received, 0% packet loss"
			if strings.Contains(line, "packets transmitted") && strings.Contains(line, "received") {
				fields := strings.Fields(line)
				for i, field := range fields {
					if field == "received," && i > 0 {
						if recv, parseErr := strconv.Atoi(fields[i-1]); parseErr == nil {
							result.PacketsRecv = recv
						}
					}
					if strings.HasSuffix(field, "%") && strings.Contains(line, "packet loss") {
						lossStr := strings.TrimSuffix(field, "%")
						if loss, parseErr := strconv.ParseFloat(lossStr, 64); parseErr == nil {
							result.PacketLoss = loss
						}
					}
				}
			}
			
			// 解析延迟统计: "round-trip min/avg/max/stddev = 1.234/2.345/3.456/0.123 ms"
			if strings.Contains(line, "round-trip") && strings.Contains(line, "=") {
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					latencyPart := strings.TrimSpace(parts[1])
					latencyValues := strings.Split(latencyPart, "/")
					if len(latencyValues) >= 2 {
						result.AvgLatency = fmt.Sprintf("%.1f ms", parseFloat(latencyValues[1]))
					}
				}
			}
		}
	}
	
	// 如果丢包率大于0，标记为部分失败
	if result.PacketLoss > 0 {
		if result.PacketLoss == 100 {
			result.Success = false
			result.ErrorMsg = "所有数据包丢失"
		} else {
			result.ErrorMsg = fmt.Sprintf("%.1f%% 数据包丢失", result.PacketLoss)
		}
	}
	
	if result.AvgLatency == "" {
		result.AvgLatency = "N/A"
	}
	
	return result
}

// parseFloat 安全解析浮点数
func parseFloat(s string) float64 {
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return 0.0
}

func RebootSystem() error {
	// 检查权限
	if os.Getuid() != 0 {
		return fmt.Errorf("需要root权限执行重启操作")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "reboot")
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("重启命令执行超时")
	}

	return err
}

func ShutdownSystem() error {
	// 检查权限
	if os.Getuid() != 0 {
		return fmt.Errorf("需要root权限执行关机操作")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "shutdown", "-h", "now")
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("关机命令执行超时")
	}

	return err
}

func RestartSystemService(serviceName string) error {
	// 检查权限
	if os.Getuid() != 0 {
		return fmt.Errorf("需要root权限重启系统服务")
	}

	// 验证服务名称
	if serviceName == "" {
		return fmt.Errorf("服务名称不能为空")
	}
	if len(serviceName) > 100 {
		return fmt.Errorf("服务名称过长")
	}
	// 防止命令注入
	if strings.ContainsAny(serviceName, "; | & $ ` ( ) [ ] { } < > ? * \\ \n \r \t") {
		return fmt.Errorf("服务名称包含非法字符")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "restart", serviceName)
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("重启服务超时")
	}

	return err
}
