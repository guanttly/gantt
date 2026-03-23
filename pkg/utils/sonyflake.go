package utils

import (
	"fmt"
	"time"

	"github.com/sony/sonyflake"
)

var settings = sonyflake.Settings{
	StartTime: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC), //起始时间
	MachineID: func() (uint16, error) { //机器ID获取函数
		return 1, nil
	},
}

var sf = sonyflake.NewSonyflake(settings)

// 获取雪花ID
func NextID() int64 {
	id, err := sf.NextID()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate Sonyflake ID: %v", err))
	}
	return int64(id)
}
