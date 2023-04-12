/***************************************************************************
 *
 * Copyright (c) 2020 Bonc.com.cn, Inc. All Rights Reserved
 *
 **************************************************************************/

/**
 * @file    pr_review.go
 * @author  qihouliang(qihouliang@bonc.com.cn)
 * @date    2023/4/12 08:16
 * @brief
 */

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/appleboy/CodeGPT/openai"
	"github.com/appleboy/CodeGPT/prompt"
	"github.com/appleboy/CodeGPT/util"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	fetchPRFilesURL   = "%s/repos/%s/%s/pulls/%d/files"
	fetchCommitsURL   = "%s/repos/%s/%s/pulls/%d/commits"
	submitCommentsURL = "%s/repos/%s/%s/pulls/%d/comments"
)

// github parameters
var (
	apiBaseurl string
	token      string
	owner      string
	repo       string

	initFlag = false
)

type PullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		URL    string `json:"url"`
		Number int    `json:"number"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type File struct {
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Patch    string `json:"patch"`
}

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

type CommentRequest struct {
	Body     string `json:"body"`
	Path     string `json:"path"`
	Position int    `json:"position"`
	CommitID string `json:"commit_id"`
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if !initFlag {
		initGithubParameter()
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing request body:", err)
		}
	}(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var event PullRequestEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		http.Error(w, "Error unmarshaling request body", http.StatusInternalServerError)
		return
	}

	if event.Action == "opened" || event.Action == "reopened" || event.Action == "synchronize" {
		log.Println("Pull request event received:", event.Action)
		owner, repo := parseRepoInfo(event.Repository.FullName)
		var files []File
		files, err = fetchPRFiles(owner, repo, event.Number)
		if err != nil {
			log.Println("Error fetching PR files:", err)
			return
		}

		commitID, err := fetchPRCommitID(owner, repo, event.Number)
		log.Println("Commit ID:", commitID)
		if err != nil {
			log.Println("Error fetching PR commit ID:", err)
			return
		}
		submitReview(owner, repo, event.Number, commitID, files)
	} else {
		log.Println("Ignored event:", event.Action)
	}
}

func initGithubParameter() {
	apiBaseurl = viper.GetString("git.api_base_url")
	token = viper.GetString("git.token")
	owner = viper.GetString("git.owner")
	repo = viper.GetString("git.repo")
	log.Printf("apiBaseurl=%s, token=%s, owner=%s, repo=%s\n", apiBaseurl, token, owner, repo)
	initFlag = true
}

func parseRepoInfo(fullName string) (owner, repo string) {
	s := strings.Split(fullName, "/")
	return s[0], s[1]
}

func fetchPRFiles(owner, repo string, prNumber int) ([]File, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf(fetchPRFilesURL, apiBaseurl, owner, repo, prNumber), nil)
	if err != nil {
		log.Println("Error creating request:", err)
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "token "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			return nil, err
		}

		var files []File
		err = json.Unmarshal(bodyBytes, &files)
		if err != nil {
			log.Println("Error unmarshaling response body:", err)
			return nil, err
		}
		return files, nil
	} else {
		log.Printf("Error fetching PR files: %d\n", resp.StatusCode)
		return nil, fmt.Errorf("Error fetching PR files: %d", resp.StatusCode)
	}
}

func fetchPRCommitID(owner, repo string, prNumber int) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf(fetchCommitsURL, apiBaseurl, owner, repo, prNumber), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing request body:", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var commits []struct {
		SHA string `json:"sha"`
	}

	err = json.Unmarshal(body, &commits)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		return "", fmt.Errorf("No commits found in PR")
	}

	return commits[len(commits)-1].SHA, nil
}

func submitReview(owner, repo string, prNumber int, commitID string, files []File) {
	for _, file := range files {
		comment, err := getChatGPTComment(file.Patch)
		if err != nil {
			comment = fmt.Sprintf("Error generating comment for file %s: %v", file.Filename, err)
			log.Printf("Error generating comment for file %s: %v\n", file.Filename, err)
		}
		position := getLastLinePosition(file.Patch)
		err = submitReviewComment(owner, repo, prNumber, commitID, file.Filename, comment, position)
		if err != nil {
			log.Printf("Error submitting review comment for file %s: %v\n", file.Filename, err)
		} else {
			log.Printf("Review comment submitted for file %s\n\n", file.Filename)
		}
	}
}

func getChatGPTComment(patch string) (string, error) {
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
		prompt.CodeReviewTemplate,
		util.Data{
			"file_diffs": patch,
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

func getLastLinePosition(patch string) int {
	lines := strings.Split(patch, "\n")
	position := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			continue
		}
		position++
	}
	return position
}

func submitReviewComment(owner, repo string, prNumber int, commitID, filePath, comment string, position int) error {
	client := &http.Client{}

	commentRequest := CommentRequest{
		Body:     comment,
		Path:     filePath,
		Position: position,
		CommitID: commitID,
	}

	jsonData, err := json.Marshal(commentRequest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf(submitCommentsURL, apiBaseurl, owner, repo, prNumber), bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating request:", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error submitting review comment:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		log.Println("Review comment submitted successfully")
	} else {
		return fmt.Errorf("error: %d", resp.StatusCode)
	}

	return nil
}
