package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"fmt"
	"time"
	"io/ioutil"
	"encoding/json"
)

var db *gorm.DB

func init() {
	//open a db connection
	var err error
	db, err = gorm.Open("mysql", "root:root@/gin-example?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic("failed to connect database")
	}

	//Migrate the schema
	db.AutoMigrate(&todoModel{})

	db.AutoMigrate(&logModel{})
}

func main() {
	router := gin.Default()

	v1 := router.Group("/api/v1/todos")
	{
		v1.POST("/", createTodo)
		v1.GET("/", fetchAllTodo)
		v1.GET("/:id", fetchSingleTodo)
		v1.PUT("/:id", updateTodo)
		v1.DELETE("/:id", deleteTodo)
	}
	router.Run()
}

type (
	// todoModel describes a todoModel type
	todoModel struct {
		gorm.Model
		Title     string `json:"title"`
		Completed int    `json:"completed"`
	}

	// transformedTodo represents a formatted todo
	transformedTodo struct {
		ID        uint   `json:"id"`
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}


	// logModel describes a logModel type
	logModel struct {
		gorm.Model
		RequestParam   string    `json:"request_param"`
		RequestURI     string    `json:"request_uri"`
		RequestMethod  string    `json:"request_method"`
		RequestTimes   string    `json:"request_time"`
		Response       string    `json:"response" gorm:"type:text`
		ResponseStatus string    `json:"response_status"`
		ResponseTimes  time.Time `json:"response_times" gorm:"column:response_times" sql:"DEFAULT:current_timestamp"`
		TotalTime      string    `json:"total_time"`
	}

	// transformedLog represents a formatted log
	transformedLog struct {
		ID             uint      `json:"id"`
		RequestParam   string    `json:"request_param"`
		RequestURI     string    `json:"request_uri"`
		RequestMethod  string    `json:"request_method"`
		RequestTimes   string    `json:"request_time"`
		Response       string    `json:"response"`
		ResponseStatus string    `json:"response_status"`
		ResponseTimes  time.Time `json:"response_times"`
		TotalTime      string    `json:"total_time"`
	}
)

// createTodo add a new todo
func createTodo(c *gin.Context) {
	completed, _ := strconv.Atoi(c.PostForm("completed"))
	todo := todoModel{Title: c.PostForm("title"), Completed: completed}
	db.Save(&todo)

	res := gin.H{"status": http.StatusCreated, "message": "Todo item created successfully!", "resourceId": todo.ID}

	response, _ := json.Marshal(res)
	logAPI(c, string(response))

	c.JSON(http.StatusCreated, res)
}

// fetchAllTodo fetch all todos
func fetchAllTodo(c *gin.Context) {
	var todos []todoModel
	var _todos []transformedTodo

	db.Find(&todos)

	if len(todos) <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No todo found!"})
		return
	}

	//transforms the todos for building a good response
	for _, item := range todos {
		completed := false
		if item.Completed == 1 {
			completed = true
		} else {
			completed = false
		}
		_todos = append(_todos, transformedTodo{ID: item.ID, Title: item.Title, Completed: completed})
	}

	res := gin.H{"status": http.StatusOK, "data": _todos}

	response, _ := json.Marshal(res)

	logAPI(c, string(response))

	c.JSON(http.StatusOK, res)
}

// fetchSingleTodo fetch a single todo
func fetchSingleTodo(c *gin.Context) {
	var todo todoModel
	todoID := c.Param("id")

	db.First(&todo, todoID)

	if todo.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No todo found!"})
		return
	}

	completed := false
	if todo.Completed == 1 {
		completed = true
	} else {
		completed = false
	}

	_todo := transformedTodo{ID: todo.ID, Title: todo.Title, Completed: completed}

	res := gin.H{"status": http.StatusOK, "data": _todo}

	response, _ := json.Marshal(res)

	logAPI(c, string(response))

	c.JSON(http.StatusOK, res)
}

// updateTodo update a todo
func updateTodo(c *gin.Context) {
	var todo todoModel
	todoID := c.Param("id")

	db.First(&todo, todoID)

	if todo.ID == 0 {
		res := gin.H{"status": http.StatusNotFound, "message": "No todo found!"}

		response, _ := json.Marshal(res)
		logAPI(c, string(response))

		c.JSON(http.StatusNotFound, res)
		return
	}

	db.Model(&todo).Update("title", c.PostForm("title"))
	completed, _ := strconv.Atoi(c.PostForm("completed"))
	db.Model(&todo).Update("completed", completed)

	res := gin.H{"status": http.StatusOK, "message": "Todo updated successfully!"}

	response, _ := json.Marshal(res)
	logAPI(c, string(response))

	c.JSON(http.StatusOK, res)
}

// deleteTodo remove a todo
func deleteTodo(c *gin.Context) {
	var todo todoModel
	todoID := c.Param("id")

	db.First(&todo, todoID)

	if todo.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No todo found!"})
		return
	}

	db.Delete(&todo)

	res := gin.H{"status": http.StatusOK, "message": "Todo deleted successfully!"}

	response, _ := json.Marshal(res)
	logAPI(c, string(response))

	c.JSON(http.StatusOK, res)
}

 
func logAPI(c *gin.Context, rsp string) {
	
	// get request body json
	getReqBody, _ := ioutil.ReadAll(c.Request.Body)
	reqBody, _ := json.Marshal(getReqBody)
	strReqBody := string(reqBody)
	fmt.Println(reqBody)

	// start := time.Now()
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery

	// Process request
	c.Next()

	end := time.Now()
	endFormat := end.Format("2006-01-02 15:04:05")
	latency := end.UnixNano() / 1000000
	strLatency := strconv.FormatInt(latency, 10)
	// clientIP := c.ClientIP()
	method := c.Request.Method
	statusCode := c.Writer.Status()
	strStatusCode := strconv.Itoa(statusCode)

	if raw != "" {
		path = path + "?" + raw
	}

	// insert log
	insertLogAPI := logModel{
		RequestParam: strReqBody,
		RequestURI: path,
		RequestMethod: method,
		RequestTimes: endFormat,
		Response: rsp,
		ResponseStatus: strStatusCode,
		ResponseTimes: end,
		TotalTime: strLatency,
	}

	db.Save(&insertLogAPI)

	return
}