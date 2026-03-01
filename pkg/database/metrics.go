package database

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolStatsCollector implements prometheus.Collector for pgxpool connection metrics.
type PoolStatsCollector struct {
	pool    *pgxpool.Pool
	service string

	acquiredConns      *prometheus.Desc
	idleConns          *prometheus.Desc
	totalConns         *prometheus.Desc
	maxConns           *prometheus.Desc
	constructingConns  *prometheus.Desc
	acquireCount       *prometheus.Desc
	acquireDuration    *prometheus.Desc
	canceledAcquires   *prometheus.Desc
	emptyAcquires      *prometheus.Desc
	newConnsCount      *prometheus.Desc
	maxLifetimeDestroy *prometheus.Desc
	maxIdleDestroy     *prometheus.Desc
}

// NewPoolStatsCollector creates a new Prometheus collector that exports pgxpool
// connection pool statistics as metrics.
func NewPoolStatsCollector(pool *pgxpool.Pool, service string) *PoolStatsCollector {
	labels := []string{"service"}
	return &PoolStatsCollector{
		pool:    pool,
		service: service,
		acquiredConns: prometheus.NewDesc(
			"db_pool_acquired_connections",
			"Number of currently acquired connections",
			labels, nil,
		),
		idleConns: prometheus.NewDesc(
			"db_pool_idle_connections",
			"Number of currently idle connections",
			labels, nil,
		),
		totalConns: prometheus.NewDesc(
			"db_pool_total_connections",
			"Total number of connections in the pool",
			labels, nil,
		),
		maxConns: prometheus.NewDesc(
			"db_pool_max_connections",
			"Maximum number of connections allowed",
			labels, nil,
		),
		constructingConns: prometheus.NewDesc(
			"db_pool_constructing_connections",
			"Number of connections currently being constructed",
			labels, nil,
		),
		acquireCount: prometheus.NewDesc(
			"db_pool_acquire_count_total",
			"Total number of connection acquires",
			labels, nil,
		),
		acquireDuration: prometheus.NewDesc(
			"db_pool_acquire_duration_seconds_total",
			"Total time spent acquiring connections in seconds",
			labels, nil,
		),
		canceledAcquires: prometheus.NewDesc(
			"db_pool_canceled_acquire_count_total",
			"Total number of canceled connection acquires",
			labels, nil,
		),
		emptyAcquires: prometheus.NewDesc(
			"db_pool_empty_acquire_count_total",
			"Total number of acquires that had to wait for a connection",
			labels, nil,
		),
		newConnsCount: prometheus.NewDesc(
			"db_pool_new_connections_total",
			"Total number of new connections created",
			labels, nil,
		),
		maxLifetimeDestroy: prometheus.NewDesc(
			"db_pool_max_lifetime_destroy_total",
			"Total connections destroyed due to max lifetime",
			labels, nil,
		),
		maxIdleDestroy: prometheus.NewDesc(
			"db_pool_max_idle_destroy_total",
			"Total connections destroyed due to max idle time",
			labels, nil,
		),
	}
}

// Describe sends the descriptors of all metrics to the provided channel.
func (c *PoolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.totalConns
	ch <- c.maxConns
	ch <- c.constructingConns
	ch <- c.acquireCount
	ch <- c.acquireDuration
	ch <- c.canceledAcquires
	ch <- c.emptyAcquires
	ch <- c.newConnsCount
	ch <- c.maxLifetimeDestroy
	ch <- c.maxIdleDestroy
}

// Collect reads current pool statistics and sends them as Prometheus metrics.
func (c *PoolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stat := c.pool.Stat()

	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(stat.AcquiredConns()), c.service)
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stat.IdleConns()), c.service)
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stat.TotalConns()), c.service)
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(stat.MaxConns()), c.service)
	ch <- prometheus.MustNewConstMetric(c.constructingConns, prometheus.GaugeValue, float64(stat.ConstructingConns()), c.service)
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(stat.AcquireCount()), c.service)
	ch <- prometheus.MustNewConstMetric(c.acquireDuration, prometheus.CounterValue, stat.AcquireDuration().Seconds(), c.service)
	ch <- prometheus.MustNewConstMetric(c.canceledAcquires, prometheus.CounterValue, float64(stat.CanceledAcquireCount()), c.service)
	ch <- prometheus.MustNewConstMetric(c.emptyAcquires, prometheus.CounterValue, float64(stat.EmptyAcquireCount()), c.service)
	ch <- prometheus.MustNewConstMetric(c.newConnsCount, prometheus.CounterValue, float64(stat.NewConnsCount()), c.service)
	ch <- prometheus.MustNewConstMetric(c.maxLifetimeDestroy, prometheus.CounterValue, float64(stat.MaxLifetimeDestroyCount()), c.service)
	ch <- prometheus.MustNewConstMetric(c.maxIdleDestroy, prometheus.CounterValue, float64(stat.MaxIdleDestroyCount()), c.service)
}

// RegisterPoolMetrics creates and registers a pgxpool metrics collector with
// the default Prometheus registry.
func RegisterPoolMetrics(pool *pgxpool.Pool, service string) {
	prometheus.MustRegister(NewPoolStatsCollector(pool, service))
}
