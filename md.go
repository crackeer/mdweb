package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/Eilick/mysql-client/common"
	"github.com/Eilick/mysql-client/database"

	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "*")

		if c.Request.Method == "OPTIONS" {
			c.JSON(http.StatusOK, "Options Request")
		}

		c.Set("content-type", "application/json")
		c.Next()
	}
}

func main() {

	common.GetConfigInstance()
	if _, err := os.Stat(*common.ImageDir); os.IsNotExist(err) {
		os.Mkdir(*common.ImageDir, os.ModePerm) //0777也可以os.ModePerm
		os.Chmod(*common.ImageDir, 0777)
	}
	router := gin.New()
	router.Use(Cors())
	router.Use(gin.Recovery())

	gin.SetMode(gin.DebugMode)
	staticWorkDir := "./assets/dist"
	box := packr.NewBox(staticWorkDir)

	if md, err := box.Find("index.html"); err == nil {
		router.GET("/", func(ctx *gin.Context) {
			ctx.Data(http.StatusOK, "text/html", md)
		})
	}
	router.GET("/css/:name", func(ctx *gin.Context) {
		name := ctx.Param("name")
		if css, err := box.Find("css/" + name); err == nil {
			ctx.Data(http.StatusOK, "text/css", css)
		} else {
			ctx.Data(http.StatusOK, "text/css", []byte(""))
		}
	})
	router.GET("/fonts/:name", func(ctx *gin.Context) {
		name := ctx.Param("name")
		if css, err := box.Find("fonts/" + name); err == nil {
			ctx.Data(http.StatusOK, "text/css", css)
		} else {
			ctx.Data(http.StatusOK, "text/css", []byte(""))
		}
	})
	router.GET("/js/:name", func(ctx *gin.Context) {
		name := ctx.Param("name")
		if css, err := box.Find("js/" + name); err == nil {
			ctx.Data(http.StatusOK, "application/javascript", css)
		} else {
			ctx.Data(http.StatusOK, "application/javascript", []byte(""))
		}
	})

	router.StaticFS("/image", http.Dir(*common.ImageDir))
	router.POST("/markdown/upload_image", uploadImage)
	router.POST("/markdown/create", CreateMd)
	router.POST("/markdown/update", UpdateMd)
	router.POST("/markdown/delete", DeleteMd)
	router.POST("/markdown/recover", RecoverMd)
	router.GET("/markdown/list", MdList)
	router.GET("/markdown/detail", SingleMd)
	router.GET("/markdown/images", getImageList)
	router.POST("/markdown/del_image", delUploadImg)

	router.Run(":" + *common.Port)

}

type Markdown struct {
	Id      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func CreateMd(ctx *gin.Context) {
	var md Markdown
	if err := ctx.BindJSON(&md); err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	id, err := database.AddArticle(md.Title, md.Content)

	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"id":   id,
	})

}

func UpdateMd(ctx *gin.Context) {
	var md Markdown
	if err := ctx.BindJSON(&md); err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	id, err := database.UpdateMd(md.Id, md.Title, md.Content)

	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"id":   id,
	})

}

func DeleteMd(ctx *gin.Context) {
	var md Markdown
	if err := ctx.BindJSON(&md); err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	d := database.SingleArticle(md.Id)
	var res bool = false
	var err error
	if len(d) > 0 && d["show_status"].(int) == -1 {
		res, err = database.DeleteMdForever(md.Id)
	} else {
		res, err = database.DeleteMd(md.Id)
	}

	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"id":   res,
	})
}

func RecoverMd(ctx *gin.Context) {
	var md Markdown
	if err := ctx.BindJSON(&md); err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	res, err := database.RecoverMd(md.Id)

	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"id":   res,
	})

}

func MdList(ctx *gin.Context) {

	s := ctx.DefaultQuery("list_type", "ok")
	list := database.ArticleList(s)

	ctx.JSON(http.StatusOK, list)

}

func SingleMd(ctx *gin.Context) {

	id := ctx.DefaultQuery("id", "0")
	d := database.SingleArticle(id)

	ctx.JSON(http.StatusOK, d)

}

func uploadImage(ctx *gin.Context) {
	header, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	dst := header.Filename
	l := strings.Split(dst, ".")
	md5Name := common.Md5Crypt([]byte(dst + common.GetNowDateTimeString()))
	fullPath := fmt.Sprintf(*common.ImageDir+"/%s.%s", md5Name, l[1])

	if err := ctx.SaveUploadedFile(header, fullPath); err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"file": fmt.Sprintf("%s.%s", md5Name, l[1]),
	})
	return
}

func getImageList(ctx *gin.Context) {
	fs, _ := ioutil.ReadDir(*common.ImageDir)
	list := []map[string]interface{}{}
	for _, file := range fs {
		t := file.ModTime()
		ts := t.Format("2006-01-02 15:04:05")
		list = append(list, map[string]interface{}{
			"name":      file.Name(),
			"create_at": ts,
			"url":       "http://" + ctx.Request.Host + "/image/" + file.Name(),
			"uri":       "/image/" + file.Name(),
		})
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"data": list,
	})
}

type Pic struct {
	Name string `json:"name"`
}

func delUploadImg(ctx *gin.Context) {
	var pic Pic
	if err := ctx.BindJSON(&pic); err != nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}

	filePath := *common.ImageDir + "/" + pic.Name
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code": -1,
			"msg":  "暂无该图片",
		})
		return
	}
	err := os.Remove(filePath)

	if err == nil {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "删除成功",
		})
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": -2,
		"msg":  "删除失败",
	})
}