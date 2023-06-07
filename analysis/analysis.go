/***************************************************************************
 *
 * Copyright (c) 2020 Bonc.com.cn, Inc. All Rights Reserved
 *
 **************************************************************************/

/**
 * @file    analysis.go
 * @author  qihouliang(qihouliang@bonc.com.cn)
 * @date    2023/6/6 16:57
 * @brief
 */

package analysis

import (
	"bufio"
	"context"
	"fmt"
	"github.com/appleboy/CodeGPT/openai"
	"github.com/appleboy/CodeGPT/prompt"
	"github.com/appleboy/CodeGPT/util"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	maxLinesPerFragment = 300
)

var (
	functionKeywords = []string{"func", "def"} // 可根据需要扩展其他编程语言的函数关键字
)

func DoAnalysis() {
	// 定义目标文件夹路径
	currentDir, err := os.Getwd()
	directoryPath := currentDir + "/target/source"
	// 获取目标文件夹下的所有文件
	files, err := os.ReadDir(directoryPath)
	if err != nil {
		fmt.Printf("无法读取目录：%s\n", err)
		return
	}

	// 创建结果文件夹
	resultDirectory := currentDir + "/target/result"
	exist := util.Exists(resultDirectory)
	if exist {
		os.RemoveAll(resultDirectory)
	}
	err = os.Mkdir(resultDirectory, 0755)
	if err != nil {
		fmt.Printf("无法创建结果文件夹：%s\n", err)
		return
	}

	// 遍历文件夹中的每个文件
	for _, file := range files {
		if file.IsDir() {
			continue // 忽略子文件夹
		}

		// 读取文件内容,解析代码片段
		filePath := filepath.Join(directoryPath, file.Name())
		codeFragments, err := splitCodeIntoFragments(filePath)

		if err != nil {
			fmt.Printf("分割代码失败：%s\n", err)
			continue
		}

		// 请求OpenAI API解析代码片段
		for _, fragment := range codeFragments {
			//log.Println("qhl, out:" + fragment)
			result, err := getChatGPTAnalysis(fragment)
			if err != nil {
				fmt.Printf("API请求错误：%s\n", err)
				continue
			}

			// 保存OpenAI返回的结果到文件
			resultFilePath := filepath.Join(resultDirectory, file.Name()+".txt")
			err = os.WriteFile(resultFilePath, []byte(result), 0644)
			if err != nil {
				fmt.Printf("无法保存结果文件：%s\n", err)
			}
			time.Sleep(time.Duration(viper.GetInt("server.sleep_time_second")) * time.Second)
		}
	}

	// 对所有源代码文件的结果进行总结
	summarizeResults(resultDirectory)
}

func splitCodeIntoFragments(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var fragments []string
	var currentFragment []string
	//var isInFunction bool

	for scanner.Scan() {
		line := scanner.Text()
		if len(currentFragment) > maxLinesPerFragment {
			fragments = append(fragments, strings.Join(currentFragment, "\n"))
			currentFragment = []string{}
		}
		currentFragment = append(currentFragment, line)

		//if isFunctionDeclaration(line) {
		//	if len(currentFragment) > 0 {
		//		fragments = append(fragments, strings.Join(currentFragment, "\n"))
		//		currentFragment = []string{}
		//	}
		//	isInFunction = true
		//}
		//
		//if isInFunction {
		//	currentFragment = append(currentFragment, line)
		//}
		//
		//if strings.TrimSpace(line) == "" && isInFunction {
		//	isInFunction = false
		//}
	}

	if len(currentFragment) > 0 {
		fragments = append(fragments, strings.Join(currentFragment, "\n"))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return fragments, nil
}

func isFunctionDeclaration(line string) bool {
	line = strings.TrimSpace(line)

	for _, keyword := range functionKeywords {
		if strings.HasPrefix(line, keyword) {
			return true
		}
	}

	if strings.HasPrefix(line, "public") || strings.HasPrefix(line, "private") || strings.HasPrefix(line, "protected") {
		// Java函数开头的访问修饰符
		return true
	}

	if strings.Contains(line, "(") && (strings.Contains(line, ")") || strings.Contains(line, "{")) {
		// 包含括号的行，可以认为是函数的开头
		return true
	}

	return false
}

// 使用OpenAI API获取代码片段的功能描述
func getChatGPTAnalysis(patch string) (string, error) {
	client, err := openai.New(
		openai.WithToken(viper.GetString("openai.api_key")),
		openai.WithModel(viper.GetString("openai.model")),
		openai.WithOrgID(viper.GetString("openai.org_id")),
		openai.WithProxyURL(viper.GetString("openai.proxy")),
		openai.WithSocksURL(viper.GetString("openai.socks")),
		openai.WithBaseURL(viper.GetString("openai.base_url")),
		openai.WithTimeout(viper.GetDuration("openai.timeout")),
		openai.WithMaxTokens(viper.GetInt("openai.max_tokens")),
		openai.WithTemperature(float32(viper.GetFloat64("openai.temperature"))),
	)
	if err != nil {
		log.Println("Error creating OpenAI client:", err)
		return "", err
	}

	out, err := util.GetTemplateByString(
		prompt.CodeAnalysisTemplate,
		util.Data{
			"code_patch": patch,
		},
	)
	if err != nil {
		log.Println("Error creating prompt:", err)
		return "", err
	}

	ctx := context.Background()
	resp, err := client.Completion(ctx, out)
	if err != nil {
		log.Println("Error completing prompt:", err)
		return "", err
	}
	summarizeMessage := resp.Content

	log.Println("PromptTokens: " + strconv.Itoa(resp.Usage.PromptTokens) +
		", CompletionTokens: " + strconv.Itoa(resp.Usage.CompletionTokens) +
		", TotalTokens: " + strconv.Itoa(resp.Usage.TotalTokens))

	// Output core review summary
	log.Println("================Review Summary====================")
	log.Println("\n" + strings.TrimSpace(summarizeMessage) + "\n\n")
	log.Println("==================================================")

	return summarizeMessage, nil
}

// 对所有结果文件进行总结
func summarizeResults(resultDirectory string) {
	// 读取结果文件夹中的所有文件
	files, err := os.ReadDir(resultDirectory)
	if err != nil {
		fmt.Printf("无法读取结果文件夹：%s\n", err)
		return
	}
	// 存储功能总结的映射
	summary := make(map[string]int)

	// 统计每个文件中的功能描述
	for _, file := range files {
		if file.IsDir() {
			continue // 忽略子文件夹
		}

		// 读取结果文件内容
		filePath := filepath.Join(resultDirectory, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("无法读取结果文件：%s\n", err)
			continue
		}

		// 统计功能描述
		description := string(content)
		summary[description]++
	}

	// 输出功能总结
	fmt.Println("源代码功能总结：")
	for description, count := range summary {
		fmt.Printf("- %s: %d\n", description, count)
	}
}
