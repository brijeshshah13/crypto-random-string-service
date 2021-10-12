package main

import (
	"fmt"
	"github.com/brijeshshah13/crypto-random-string"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	r := gin.Default()
	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/random-string", func(ctx *gin.Context) {
		// just a practice on how to capture canceled requests
		select {
		case <-ctx.Request.Context().Done():
			fmt.Println(ctx.Request.Context().Err())
			return
		default:
		}
		generator := cryptorandomstring.New()
		if str, err := generator.WithLength(10).Generate(); err != nil {
			fmt.Println(err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"string": str,
			})
		}
	})
	r.Run()
}
