package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

// API 端点
const (
	baseURL          = "http://lgxt.wutp.com.cn/api"
	loginAPI         = baseURL + "/login"
	userInfoAPI      = baseURL + "/userInfo"
	myCoursesAPI     = baseURL + "/myCourses"
	myCourseWorksAPI = baseURL + "/myCourseWorks"
	submitAnswerAPI  = baseURL + "/submitAnswer"
)

// User 用户数据
type User struct {
	Name      string
	StudentNo string
}

// Course 课程数据
type Course struct {
	CourseID   int
	CourseName string
}

// CourseWork 作业数据
type CourseWork struct {
	WorkID   int
	WorkName string
}

// Config 配置文件数据
type Config struct {
	Account   string `json:"account"`
	Password  string `json:"password"`
	SleepTime int    `json:"sleeptime"`
}

// Client API客户端
type Client struct {
	httpClient    *http.Client
	authorization string
}

// newClient 创建新的API客户端
func newClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

// login 进行登录并获取授权令牌
func (c *Client) login(account, password string) error {
	// 构造表单
	formData := url.Values{
		"loginName": {account},
		"password":  {password},
	}

	// 发送请求
	respBody, err := c.sendPostRequest(loginAPI, formData, "")
	if err != nil {
		return err
	}

	// 提取令牌
	data, err := parseJSON(respBody, "data")
	if err != nil {
		return fmt.Errorf("无法提取 Authorization 值: %v", err)
	}
	authToken, ok := data.(string)
	if !ok {
		return fmt.Errorf("Authorization 格式错误")
	}

	c.authorization = authToken
	return nil
}

// getUserInfo 获取用户信息
func (c *Client) getUserInfo() (User, error) {
	var user User

	// 发送请求
	respBody, err := c.sendPostRequest(userInfoAPI, nil, c.authorization)
	if err != nil {
		return user, err
	}

	// 解析
	data, err := parseJSON(respBody, "data")
	if err != nil {
		return user, err
	}

	// 提取用户信息
	userInfo, ok := data.(map[string]interface{})
	if !ok {
		return user, fmt.Errorf("返回值错误")
	}
	userName, ok := userInfo["userName"].(string)
	if !ok {
		return user, fmt.Errorf("无法提取 userName 值")
	}
	studentNo, ok := userInfo["studentNo"].(string)
	if !ok {
		return user, fmt.Errorf("无法提取 studentNo 值")
	}

	user.Name = userName
	user.StudentNo = studentNo
	return user, nil
}

// getCourses 获取课程信息
func (c *Client) getCourses() ([]Course, error) {
	// 发送请求
	respBody, err := c.sendPostRequest(myCoursesAPI, nil, c.authorization)
	if err != nil {
		return nil, err
	}

	// 解析
	data, err := parseJSON(respBody, "data")
	if err != nil {
		return nil, err
	}

	// 转换为课程列表
	courseList, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("数据格式错误")
	}

	var courses []Course
	for _, item := range courseList {
		courseMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		courseID, ok1 := courseMap["courseId"].(float64)
		courseName, ok2 := courseMap["courseName"].(string)
		if ok1 && ok2 {
			courses = append(courses, Course{
				CourseID:   int(courseID),
				CourseName: courseName,
			})
		}
	}

	return courses, nil
}

// getCourseWorks 获取作业信息
func (c *Client) getCourseWorks(courseID int) ([]CourseWork, error) {
	// 构造表单
	formData := url.Values{
		"courseId": {strconv.Itoa(courseID)},
	}

	// 发送请求
	respBody, err := c.sendPostRequest(myCourseWorksAPI, formData, c.authorization)
	if err != nil {
		return nil, err
	}

	// 解析
	data, err := parseJSON(respBody, "data")
	if err != nil {
		return nil, err
	}

	// 转换为作业列表
	workList, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("数据格式错误")
	}

	var works []CourseWork
	for _, item := range workList {
		workMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		workID, ok1 := workMap["workId"].(float64)
		workName, ok2 := workMap["workName"].(string)
		if ok1 && ok2 {
			works = append(works, CourseWork{
				WorkID:   int(workID),
				WorkName: workName,
			})
		}
	}

	return works, nil
}

// submitAnswer 提交作业答案
func (c *Client) submitAnswer(workID, grade int) error {
	// 构造表单
	formData := url.Values{
		"workId": {strconv.Itoa(workID)},
		"grade":  {strconv.Itoa(grade)},
	}

	// 发送请求
	_, err := c.sendPostRequest(submitAnswerAPI, formData, c.authorization)
	if err != nil {
		return err
	}

	return nil
}

// sendPostRequest 发送POST请求
func (c *Client) sendPostRequest(apiURL string, formData url.Values, authorization string) ([]byte, error) {
	// 构造请求体
	var requestBody *bytes.Reader
	if formData != nil {
		requestBody = bytes.NewReader([]byte(formData.Encode()))
	} else {
		requestBody = bytes.NewReader(nil)
	}

	// 创建请求
	req, err := http.NewRequest("POST", apiURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP 状态码: %d, 响应内容: %s", resp.StatusCode, string(body))
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取失败: %v", err)
	}
	return body, nil
}

// clearScreen 清屏
func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls") // Windows
	} else {
		cmd = exec.Command("clear") // Unix/Linux/Mac
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// parseJSON 解析 JSON
func parseJSON(body []byte, key string) (interface{}, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("无法解析 JSON: %v", err)
	}
	value, ok := parsed[key]
	if !ok {
		return nil, fmt.Errorf("JSON 中未找到键: %s", key)
	}
	return value, nil
}

// loadConfig 从配置文件加载账号密码
func loadConfig() (Config, error) {
	var config Config

	// 检查配置文件是否存在
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		return config, fmt.Errorf("配置文件不存在")
	}

	// 读取配置文件
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		return config, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析 JSON
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return config, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return config, nil
}

// saveAccountConfig 保存账号密码到配置文件
func saveAccountConfig(account, password string) error {
	config, err := loadConfig()
	if err != nil {
		config = Config{}
	}
	config.Account = account
	config.Password = password

	// 转换为 JSON
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("生成配置文件失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile("config.json", configJSON, 0600)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// saveSleepConfig 保存延迟设定到配置文件
func saveSleepConfig(sleepTime int) error {
	config, err := loadConfig()
	if err != nil {
		config = Config{}
	}
	config.SleepTime = sleepTime

	// 转换为 JSON
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("生成配置文件失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile("config.json", configJSON, 0600)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// logout 退出登录，清空配置文件中的账号密码
func logout() error {
	// 清空账号密码
	err := saveAccountConfig("", "")
	if err != nil {
		return fmt.Errorf("退出登录失败: %v", err)
	}
	return nil
}

func main() {
	var account, password string

	// 尝试从配置文件加载账号密码
	config, err := loadConfig()
	if err == nil && config.Account != "" && config.Password != "" {
		// 配置文件存在且解析成功
		account = config.Account
		password = config.Password
		fmt.Println("已从配置文件加载账号信息")
	} else {
		// 配置文件不存在或解析失败，需要用户输入
		fmt.Print("账号: ")
		fmt.Scanln(&account)
		fmt.Print("密码: ")
		fmt.Scanln(&password)

		// 清屏
		clearScreen()

		// 保存账号密码到配置文件
		err = saveAccountConfig(account, password)
		if err != nil {
			fmt.Printf("保存配置文件失败: %v\n", err)
		} else {
			fmt.Println("账号信息已保存到配置文件")
		}

		// 延迟1秒
		time.Sleep(1 * time.Second)
	}

	// 创建客户端
	client := newClient()

	// 登录
	err = client.login(account, password)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		fmt.Println("请重新登录")
		err = logout()
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		// 延迟2秒
		time.Sleep(2 * time.Second)

		// 重新启动程序
		main()
		return
	}

	// 获取用户信息
	user, err := client.getUserInfo()
	if err != nil {
		fmt.Printf("获取用户信息失败: %v\n", err)
		return
	}
	fmt.Printf("登录成功！你好，%s（学号：%s）\n\n", user.Name, user.StudentNo)

	// 获取课程信息
	courses, err := client.getCourses()
	if err != nil {
		fmt.Printf("获取课程信息失败: %v\n", err)
		return
	}

	// 展示课程信息
	fmt.Println("你的课程: ")
	for _, course := range courses {
		fmt.Printf("%s [%d]\n", course.CourseName, course.CourseID)
	}

	// 选择课程
	fmt.Println("\n自动提交所有课程的所有作业 [-1]")
	fmt.Println("退出登录 [0]")
	fmt.Print("\n请输入课程ID: ")

	var selectedCourseID int
	_, err = fmt.Scanln(&selectedCourseID)
	if err != nil {
		fmt.Printf("输入错误: %v\n", err)
		return
	}

	// 输入0退出登录
	if selectedCourseID == 0 {
		err = logout()
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		// 重新启动程序
		fmt.Println("已退出登录")

		// 延迟2秒
		time.Sleep(2 * time.Second)

		clearScreen()
		main()
		return
	}

	// 输入-1循环所有课程和作业
	if selectedCourseID == -1 {
		fmt.Println("\n开始自动提交所有课程的所有作业...")

		// 尝试从配置文件加载延迟时间
		var sleepTime int
		config, err := loadConfig()
		if err == nil && config.SleepTime != 0 {
			// 配置文件存在且解析成功
			sleepTime = config.SleepTime
		} else {
			// 配置文件不存在或解析失败，需要用户输入
			fmt.Print("请设置两次自动提交的延迟(秒，默认60): ")
			_, err := fmt.Scanln(&sleepTime)
			if err != nil {
				sleepTime = 60
			}

			// 保存延迟设定到配置文件
			err = saveSleepConfig(sleepTime)
			if err != nil {
				fmt.Printf("保存配置文件失败: %v\n", err)
			} else {
				fmt.Println("延迟设定已保存到配置文件")
			}

			// 延迟1秒
			time.Sleep(1 * time.Second)
		}

		fmt.Printf("每次提交间隔 %d 秒...", sleepTime)

		for _, course := range courses {
			fmt.Printf("\n正在处理课程: %s [%d]\n", course.CourseName, course.CourseID)

			// 获取课程作业
			works, err := client.getCourseWorks(course.CourseID)
			if err != nil {
				fmt.Printf("获取课程 [%d] 作业失败: %v\n", course.CourseID, err)
				continue
			}

			// 提交该课程的所有作业
			for _, work := range works {
				fmt.Printf("  提交作业: %s [%d]...", work.WorkName, work.WorkID)
				err = client.submitAnswer(work.WorkID, 100)
				if err != nil {
					fmt.Printf("失败: %v\n", err)
				} else {
					fmt.Printf("成功!\n")
				}

				// 休息一会
				time.Sleep(time.Duration(sleepTime) * time.Second)
			}
			fmt.Println()
		}

		fmt.Println("\n所有作业提交完成!")
		fmt.Println("按任意键返回课程列表，按0退出...")

		// 使用 bufio 读取用户输入
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadByte()
		if input == '0' {
			return
		} else {
			// 清屏
			clearScreen()
			main()
		}
		return
	}

	// 获取对应的作业信息
	works, err := client.getCourseWorks(selectedCourseID)
	if err != nil {
		fmt.Printf("获取作业信息失败: %v\n", err)
		return
	}

	// 展示作业信息
	fmt.Println("\n作业列表：")
	for _, work := range works {
		fmt.Printf("%s [%d]\n", work.WorkName, work.WorkID)
	}
	fmt.Println("\n返回课程列表 [0]")

	// 选择作业
	fmt.Print("\n请输入作业ID: ")
	var selectedWorkID int
	_, err = fmt.Scanln(&selectedWorkID)
	if err != nil {
		fmt.Printf("输入错误: %v\n", err)
		return
	}

	// 输入0返回课程列表
	if selectedWorkID == 0 {
		// 清屏
		clearScreen()
		main()
		return
	}

	// 提交答案
	err = client.submitAnswer(selectedWorkID, 100) // 满昏!
	if err != nil {
		fmt.Printf("提交失败: %v\n", err)
		return
	}

	fmt.Println("\n提交成功！")

	// 等待用户输入
	fmt.Println("按任意键返回课程列表，按0退出...")

	// 使用 bufio 读取用户输入
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadByte()
	if input == '0' {
		return
	} else {
		// 清屏
		clearScreen()
		main()
	}
}
