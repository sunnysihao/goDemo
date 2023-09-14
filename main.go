package main

import (
	"demo/utils"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var lock = sync.Mutex{}
var wg sync.WaitGroup
var n = 0

func async() {

	defer wg.Done()
	//lock.Lock()
	for i := 0; i < 10000; i++ {
		n++
	}
	log.Println("n的值为:", n)
	//lock.Unlock()
}

// 中间件
func myHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.Request.Header.Get("Authorization")
		if token != "secret-token" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			ctx.Abort() // 阻止后续的处理器执行
			return
		}
		ctx.Set("session", "session111") // set设置中间件中的值
		ctx.Next()                       // 执行后续操作

	}
}

func redir(ctx *gin.Context) {
	ctx.Redirect(307, "/form")
}

func read(client *gorm.DB, city string) *User {
	var user User // First取第一个，Take取查询到的任意一个
	//user.Id = 888 //先给user的成员变量赋值，相当于在where条件中加了id=888 and city=？
	err := client.Select("id, city").Where("city=?", city).Limit(1).First(&user).Error
	checkErr(err)
	return &user
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var ClientPool *gorm.DB

func main() {
	dataSourceName := "sunny:Zsh_9519608@tcp(43.136.235.149:3306)/go_db?charset=utf8&parseTime=true"
	ClientPool, err := gorm.Open(mysql.Open(dataSourceName), nil)
	checkErr(err)
	DB, err := ClientPool.DB()
	DB.SetMaxIdleConns(10)  //设置初始连接数，最大空闲连接数
	DB.SetMaxOpenConns(200) //设置最大打开连接数
	defer DB.Close()
	go func() {
		err := http.ListenAndServe("localhost:8090", nil)
		if err != nil {
			return
		}
	}()
	router := gin.Default()
	//router.Use(myHandler())
	router.GET("/get", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello")
	})

	// 写数据库
	router.POST("adduser", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		err := ClientPool.Create(&user).Error //插入一条记录，如果一次插入多条记录，传入切片，使用BatchCreate
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		c.JSON(200, gin.H{"msg": "OK"})
	})

	//删除数据
	router.DELETE("deluser", func(c *gin.Context) {
		id := c.Query("id")
		var user User
		err = ClientPool.Where("id=?", id).First(&user).Error
		if err != nil {
			c.JSON(200, gin.H{"code": "ERROR", "msg": err.Error()})
			return
		} else {
			err := ClientPool.Where("id=?", id).Delete(User{}).Error
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
			}
			c.JSON(200, gin.H{"msg": "OK"})
		}

	})
	// 路由参数
	router.GET("/user/:userid/*action", func(ctx *gin.Context) { // *action 表示多个参数
		id := ctx.Param("userid")
		action := ctx.Param("action")
		//截取/
		action = strings.Replace(action, "/", " | ", -1)
		ctx.String(http.StatusOK, id+" is "+action)
	})
	// 查询参数
	router.GET("/user", func(ctx *gin.Context) {
		name := ctx.Query("name")
		n := ctx.Query("n")
		m, _ := strconv.Atoi(n)
		fmt.Println(reflect.TypeOf(name))
		if len(name) == 0 {
			r := 1 / m
			fmt.Printf("查询参数错误: %d\n", r)
		}
		ctx.String(http.StatusOK, fmt.Sprintf("hello %s", name))
	})
	// 返回json
	router.POST("/user", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"a": 1, "b": gin.H{"a1": 111},
		})
	})
	router.POST("/json", func(ctx *gin.Context) {
		jc, _ := ctx.GetRawData()
		var m map[string]interface{}
		_ = json.Unmarshal(jc, &m)
		ctx.JSON(200, m)
		jsonData, _ := json.MarshalIndent(m, "", "    ") // map转为格式化的json字符串
		fmt.Println(string(jsonData))
	})
	router.POST("/form", myHandler(), func(ctx *gin.Context) {
		form := ctx.PostForm("f1")
		ctx.String(200, form)
	})
	// 重定向
	router.POST("/redirct", redir)
	// 404
	router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(404, gin.H{
			"code":    "ERROR",
			"message": "NOT FOUND",
		})
	})
	// 路由组
	routerGroup := router.Group("/user")
	{
		routerGroup.GET("/id", myHandler(), func(ctx *gin.Context) {
			session := ctx.MustGet("session").(string)

			ctx.String(200, fmt.Sprintf("user的id | %s", session))
			log.Println(session)
		})
		routerGroup.POST("/test", func(c *gin.Context) {
			jc, _ := c.GetRawData()
			type Js struct {
				A struct {
					A1 string `json:"a1"`
				} `json:"a"`
				B []int `json:"b"`
				C int   `json:"c"`
			}
			rps := struct {
				A string `json:"a"`
				B string `json:"b"`
				C string `json:"c"`
			}{}
			var js Js
			_ = json.Unmarshal(jc, &js)
			a := js.A.A1
			b := js.B
			cc := js.C
			var as utils.AStruct
			as.Val = cc
			switch a {
			case "a1":
				rps.A = a
			case "a":
				c.Redirect(301, "/get")
				return
			}

			switch {
			case len(b) > 3:
				rps.B = "大于"
			case len(b) < 3:
				rps.B = "小于"
			case len(b) == 3:
				rps.B = "等于"
			}
			ccc := as.Add()
			println(utils.Cfg, ccc)
			rps.C = fmt.Sprintf("%d", as.Add())
			//c.JSON(200, rps)
			c.JSON(200, gin.H{
				"a": rps.A,
				"b": rps.B,
				"c": as.Add(),
			})
		})
	}
	router.GET("/xiecheng", func(c *gin.Context) {
		wg.Add(3)
		for i := 0; i < 3; i++ {
			go async()
		}
		wg.Wait()
		c.String(200, "已完成====>:%d", n)
	})

	_ = router.Run()

}

type User struct { // gorm中默认User结构体对应的表格名字是users
	//gorm.Model
	Id      int    `gorm:"primary_key;auto_increment" json:"-"` //默认Id就是主键，假设要指定aid为主键，则可以加上`gorm:"colomn:aid,primarykey"`
	Keyword string `gorm:"column:keywords" json:"keyword"`      //指定对应于表格里的列名
	City    string
}

// 显示指定表格的名字
func (User) TableName() string {
	return "user"
}
