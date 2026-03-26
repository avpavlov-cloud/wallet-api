package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetVolumeReportHandler возвращает данные из Materialized View
func (s *Server) GetVolumeReportHandler(c *gin.Context) {
	// Читаем из снимка (мгновенно)
	rows, err := s.DbPool.Query(c.Request.Context(),
		"SELECT currency, total_volume, transaction_count, last_updated FROM daily_volume_report")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "отчет временно недоступен"})
		return
	}
	defer rows.Close()

	var reports []gin.H
	for rows.Next() {
		var curr string
		var vol float64
		var count int
		var updated time.Time
		if err := rows.Scan(&curr, &vol, &count, &updated); err == nil {
			reports = append(reports, gin.H{
				"currency": curr,
				"volume":   vol,
				"count":    count,
				"updated":  updated,
			})
		}
	}
	c.JSON(http.StatusOK, reports)
}
