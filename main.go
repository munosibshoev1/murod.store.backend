package main

import (
	"backend/config"
	"backend/routes"
	"fmt"
	"log"
	"os"
	"time"

	"backend/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
    
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "backend/middleware"
)

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatalf("Error loading .env file")
    }

    gin.SetMode(gin.ReleaseMode)
    log.Printf("Running in %s mode", gin.Mode())

    r := gin.Default()

    middleware.InitMetrics()
    r.Use(middleware.PrometheusMiddleware())

// /metrics endpoint
    r.GET("/metrics", func(c *gin.Context) {
    if c.ClientIP() != "172.19.0.18" {
        c.AbortWithStatus(403)
        return
    }
    promhttp.Handler().ServeHTTP(c.Writer, c.Request)
})
    // Настройка CORS
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"https://murod.store", "https://bp.murod.store"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Cashier-ID"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
    }))

    // Настройка временной зоны и планировщика задач
    location, err := time.LoadLocation("Asia/Yekaterinburg")
    if err != nil {
        fmt.Println("Ошибка загрузки временной зоны:", err)
        return

    }
    s := gocron.NewScheduler(location)
    s.Every(1).Day().At("01:01").Do(utils.CheckInstallmentRates)
    s.StartAsync() // Запуск планировщика в фоновом режиме
    // Подключение к базе данных и инициализация маршрутов
    config.ConnectDatabase()
    routes.InitializeRoutes(r)

    // Запуск сервера
    port := os.Getenv("PORT")
    if port == "" {
        port = "1414"
    }

    r.Run(":" + port)
}
