package session_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/session")
	{
		g.GET("/sign_in", SignInAction)
		g.POST("/create", CreateAction)
		g.GET("/destroy", DestroyAction)
	}
}
