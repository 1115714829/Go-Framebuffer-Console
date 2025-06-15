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
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	
	var result []NetworkInterface
	for _, iface := range interfaces {
		ni := NetworkInterface{
			Name:   iface.Name,
			Status: "down",
		}
		
		if iface.Flags&net.FlagUp != 0 {
			ni.Status = "up"
		}
		
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ipnet.IP.To4() != nil {
						ni.IPv4 = ipnet.IP.String()
					} else if ipnet.IP.To16() != nil {
						ni.IPv6 = ipnet.IP.String()
					}
				}
			}
		}
		
		result = append(result, ni)
	}
	
	return result, nil
}

type NetworkInterface struct {
	Name   string
	Status string
	IPv4   string
	IPv6   string
}

func TestNetworkConnectivity() (bool, error) {
	return TestNetworkConnectivityWithTimeout(10 * time.Second)
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