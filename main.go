/***************************************************************************
 *
 * Copyright (c) 2020 Bonc.com.cn, Inc. All Rights Reserved
 *
 **************************************************************************/

/**
 * @file    main_test.go.go
 * @author  qihouliang(qihouliang@bonc.com.cn)
 * @date    2023/4/11 22:26
 * @brief
 */

package main

import (
	"fmt"
	"github.com/appleboy/CodeGPT/cmd"
	"github.com/appleboy/CodeGPT/webhook"
	"github.com/spf13/viper"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	cmd.InitConfig()
	http.HandleFunc("/payload", webhook.HandleWebhook)
	ip := viper.GetString("server.ip")
	port := viper.GetString("server.port")
	var address = fmt.Sprintf("%s:%s", ip, port)
	fmt.Println("Listening for GitHub Webhooks on", address)
	http.ListenAndServe(address, nil)

}
