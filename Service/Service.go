package service

import (
	"Distributed-Health-Monitoring/Repository"
	"Distributed-Health-Monitoring/cache"
	"Distributed-Health-Monitoring/config"
	"Distributed-Health-Monitoring/models"
	"context"
	"errors"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Engine struct {
	Repo   Repository.IRepository
	router *gin.Engine
	Cnfg   *config.Config
}

func NewEngine() (*Engine, error) {

	cnfg, err := config.LoadConfig("config.json")
	if err != nil {
		return nil, err
	}

	db, err := config.ConnectPostgres(cnfg)
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&models.ExternalService{}, &models.ServiceCheckLog{})

	log.Println(cnfg.PostgreSQL.Database + "DATABASE " + "CONNECTED ")

	NuRepository := Repository.NewRepository(db)

	if NuRepository == nil {
		return nil, errors.New("repository is nil")
	}

	ginEngine := gin.Default()

	

	return &Engine{
		Repo:   NuRepository,
		router: ginEngine,
		Cnfg:   cnfg,
	}, nil
}

func (e *Engine) RegisterService(c *gin.Context) {

	defer func() {
		if r := recover(); r != nil {
			c.JSON(500, gin.H{"error": r})
		}
	}()

	var service *models.ExternalService

	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if service == nil {
		c.JSON(400, gin.H{"error": "service is nil"})
		return
	}

	err := e.Repo.RegisterService(c.Request.Context(), service)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	cache.MapExternalServices[service.ID] = service

	c.JSON(201, gin.H{"message": "service registered successfully", "service": service})
}

func (e *Engine) Run() error {

	addr := config.GetServerAddress(e.Cnfg)

	return e.router.Run(addr)
}

func (e *Engine) SetupRoutes() {

	// Health check
	e.router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// health-app group
	health := e.router.Group("/health-app")
	{
		// External services routes
		externalServices := health.Group("/externalServices")
		externalServices.Use(BasicAuthMiddleware(e.Cnfg.Auth))
		{
			externalServices.POST("/register", e.RegisterService)
			externalServices.GET("/list", e.ListServices)
		}

		// Health check logs routes
		healthLogs := health.Group("/healthLogs")
		{
			healthLogs.GET("/:serviceId", e.GetHealthCheckLogs)
		}
	}

	// WebSocket endpoint for live updates
	e.router.GET("/ws", e.HandleWebSocket)
}

// Or get the router to set up routes
func (e *Engine) Router() *gin.Engine {
	return e.router
}

func BasicAuthMiddleware(cfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		if !ok || user != cfg.Username || pass != cfg.Password {
			c.AbortWithStatusJSON(401, gin.H{
				"error": "unauthorized",
			})
			return
		}
		c.Next()
	}
}


func (e *Engine) ListServices(c *gin.Context) {
	services, err := e.Repo.GetAllServices(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"services": services})
}

func (e *Engine) GetHealthCheckLogs(c *gin.Context) {
	serviceID := c.Param("serviceId")
	limit := c.Query("limit")
	offset := c.Query("offset")

	id, err := strconv.ParseUint(serviceID, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid service id"})
		return
	}

	limitInt := 100
	offsetInt := 0

	if limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			limitInt = l
		}
	}

	if offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			offsetInt = o
		}
	}

	logs, err := e.Repo.GetServiceCheckLogs(c.Request.Context(), uint(id), limitInt, offsetInt)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"logs": logs})
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (configure as needed)
	},
}

func (e *Engine) HandleWebSocket(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] upgrade_failed err=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to upgrade websocket"})
		return
	}

	client := &models.Client{
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	GlobalHub.register <- client

	go func() {
		defer func() {
			GlobalHub.unregister <- client
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WS] read_error err=%v", err)
				}
				return
			}
			_ = message // Handle ping/pong if needed
		}
	}()

	go func() {
		for message := range client.Send {
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[WS] write_error err=%v", err)
				return
			}
		}
	}()
}

func (e *Engine) Scheduler(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SCHEDULER] panic: %v\n%s", r, debug.Stack())
		}
	}()

	sched, err := e.NewScheduler(e.Cnfg)
	if err != nil {
		return err
	}
	defer sched.Close()

	log.Println("[SCHEDULER] started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[SCHEDULER] stopped")
			return nil

		case <-ticker.C:
			services, err := e.Repo.GetAllServices(ctx)
			if err != nil {
				log.Println("[SCHEDULER] fetch services failed:", err)
				continue
			}

			now := time.Now()

			for _, s := range services {
				if !shouldRun(s, now) {
					continue
				}

				job := HealthCheckJob{
					ServiceName: s.Name,
					URL:         s.URL,
					Method:      s.HTTPMethod,
					Timeout:     time.Duration(s.TimeoutSeconds) * time.Second,
				}

				if err := sched.Schedule(job); err != nil {
					log.Printf(
						"[SCHEDULER] schedule_failed service=%s err=%v",
						s.Name,
						err,
					)
				}
			}
		}
	}
}

func shouldRun(s *models.ExternalService, now time.Time) bool {
	if s.LastCheckedAt == nil {
		return true
	}

	next := s.LastCheckedAt.Add(time.Duration(s.Interval) * time.Second)
	return now.After(next)
}