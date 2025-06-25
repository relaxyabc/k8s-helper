package dao

import (
	"gorm.io/gorm"
)

// ClusterInfo 表示集群信息
// 包含集群名和 IP
// 用于 clusters 表的查询结果映射
type ClusterInfo struct {
	ClusterName string `json:"cluster_name"`
	IP          string `json:"ip"`
}

// GetClusterInfos 查询所有集群名和IP
// 参数 db: gorm.DB 实例
// 返回值: ClusterInfo 切片和错误信息
func GetClusterInfos(db *gorm.DB) ([]ClusterInfo, error) {
	var clusters []struct {
		ClusterName string `gorm:"column:cluster_name"`
		IP          string `gorm:"column:ip"`
	}
	err := db.Table("clusters").Select("cluster_name, ip").Find(&clusters).Error
	if err != nil {
		return nil, err
	}
	var result []ClusterInfo
	for _, c := range clusters {
		result = append(result, ClusterInfo{ClusterName: c.ClusterName, IP: c.IP})
	}
	return result, nil
}
