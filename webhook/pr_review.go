/***************************************************************************
 *
 * Copyright (c) 2020 Bonc.com.cn, Inc. All Rights Reserved
 *
 **************************************************************************/

/**
 * @file    pr_review.go
 * @author  qihouliang(qihouliang@bonc.com.cn)
 * @date    2023/4/8 22:20
 * @brief
 */

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type WebhookPayload struct {
	Action string `json:"action"`
	PR     struct {
		Number int `json:"number"`
	} `json:"pull_request"`
}

func getPullRequestDiff(owner, repo string, number int, token string) (string, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	diffURL := fmt.Sprintf("repos/%s/%s/pulls/%d.diff", owner, repo, number)
	req, err := client.NewRequest("GET", diffURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(ctx, req, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	diffBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(diffBytes), nil
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	payload := &WebhookPayload{}
	err := json.NewDecoder(r.Body).Decode(payload)
	if err != nil {
		http.Error(w, "Failed to decode payload", http.StatusBadRequest)
		return
	}

	if payload.Action == "opened" || payload.Action == "synchronize" {
		log.Printf("Received PR #%d %s event", payload.PR.Number, payload.Action)
		diff, err := getPullRequestDiff("owner", "repo", payload.PR.Number, "your_access_token")
		if err != nil {
			log.Printf("Failed to get diff for PR #%d: %v", payload.PR.Number, err)
			return
		}

		// 处理diff获取本次提交的代码修改
		// ...

		fmt.Println(diff)
	}
}
