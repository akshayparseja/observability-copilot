package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"observability-copilot/pkg/scanner"
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/copilot?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}
	
	fmt.Println("✅ Connected to Postgres")
}

func main() {
	defer db.Close()
	
	router := gin.Default()
	
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	router.POST("/api/v1/imports", func(c *gin.Context) {
		var req struct {
			GitHubURL string `json:"github_url"`
		}
		c.BindJSON(&req)
		
		parts := strings.Split(req.GitHubURL, "/")
		repoID := parts[len(parts)-1]
		repoID = strings.TrimSuffix(repoID, ".git")
		
		result, err := scanner.ScanRepo(req.GitHubURL, repoID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		
		db.Exec(
			"INSERT INTO repos (id, name, github_url) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING",
			repoID, repoID, req.GitHubURL,
		)
		
		for _, svc := range result.Services {
			serviceID := fmt.Sprintf("%s-%s", repoID, svc)
			db.Exec(
				"INSERT INTO services (id, repo_id, name, framework, has_metrics, has_otel) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING",
				serviceID, repoID, svc, result.Framework, result.HasMetrics, result.HasOTel,
			)
		}
		
		c.JSON(200, gin.H{
			"message": "Scan complete",
			"repo_id": repoID,
			"result":  result,
		})
	})
	
	router.GET("/api/v1/repos/:repo_id/plan", func(c *gin.Context) {
		repoID := c.Param("repo_id")
		
		rows, err := db.Query(
			"SELECT name, framework, has_metrics, has_otel FROM services WHERE repo_id = $1",
			repoID,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		
		services := []map[string]interface{}{}
		for rows.Next() {
			var name, framework string
			var hasMetrics, hasOtel bool
			rows.Scan(&name, &framework, &hasMetrics, &hasOtel)
			services = append(services, map[string]interface{}{
				"name":        name,
				"framework":   framework,
				"has_metrics": hasMetrics,
				"has_otel":    hasOtel,
			})
		}
		
		c.JSON(200, gin.H{
			"repo_id":  repoID,
			"services": services,
		})
	})
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	
	fmt.Printf("🚀 Server on :%s\n", port)
	router.Run(":" + port)
}
