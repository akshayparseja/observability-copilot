
	"github.com/prometheus/client_golang/prometheus/promhttp"


// Expose Prometheus metrics endpoint
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
