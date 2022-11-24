package initial

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"wayne/src/backend/client"
)

func InitClient() {
	// 定期更新client, 5s执行一次 client.BuildApiserverClient
	go wait.Forever(client.BuildApiserverClient, 5*time.Second)
}
