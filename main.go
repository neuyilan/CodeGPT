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
	"github.com/appleboy/CodeGPT/util"
	"github.com/appleboy/CodeGPT/webhook"
	"log"
	"net/http"
)

const (
	port = "8080"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	ip, err := util.GetClientIp()
	if err != nil {
		log.Println("Can not get client ip address.")
	}
	cmd.InitConfig()
	http.HandleFunc("/payload", webhook.HandleWebhook)
	var address = fmt.Sprintf("%s:%s", ip, port)
	fmt.Println("Listening for GitHub Webhooks on", address)
	http.ListenAndServe(address, nil)
}
