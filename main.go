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
	"github.com/appleboy/CodeGPT/analysis"
	"github.com/appleboy/CodeGPT/cmd"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	cmd.InitConfig()
	analysis.DoAnalysis()
}
