// kafka-bench sends a burst of OrderCommand messages to the matching engine's
// input topic and measures end-to-end throughput + latency by consuming the
// trade.matched output topic.
//
// Designed to run inside the staging compose network (or against any reachable
// broker):
//
//	docker run --rm --network funnyoption-staging_default \
//	  -v /tmp/funnyoption-bench/kafka-bench:/kafka-bench \
//	  --entrypoint /kafka-bench alpine \
//	  --brokers kafka:9092 --prefix funnyoption.staging. \
//	  --orders 20000 --seed-levels 20
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	sharedkafka "funnyoption/internal/shared/kafka"

	kafkago "github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
)

type config struct {
	brokers       []string
	prefix        string
	market        int64
	outcome       string
	makerUser     int64
	takerUser     int64
	seedLevels    int
	seedQty       int64
	orders        int
	basePrice     int64
	priceStep     int64
	concurrency   int
	drainTimeout  time.Duration
	consumerGroup string
	label         string
	noMatch       bool
	pgDSN         string
}

func parseFlags() config {
	var brokersCSV string
	cfg := config{}

	flag.StringVar(&brokersCSV, "brokers", "kafka:9092", "comma-separated kafka brokers")
	flag.StringVar(&cfg.prefix, "prefix", "funnyoption.staging.", "kafka topic prefix (with trailing dot)")
	flag.Int64Var(&cfg.market, "market", 999000, "market id to use (pick something that won't clash with real markets)")
	flag.StringVar(&cfg.outcome, "outcome", "YES", "outcome side (YES/NO)")
	flag.Int64Var(&cfg.makerUser, "maker-user", 999001, "maker user id")
	flag.Int64Var(&cfg.takerUser, "taker-user", 999002, "taker user id (must differ from maker to avoid STP)")
	flag.IntVar(&cfg.seedLevels, "seed-levels", 20, "number of resting ask price levels to seed before blasting")
	flag.Int64Var(&cfg.seedQty, "seed-qty", 10_000_000, "quantity per seed ask (each ask absorbs this many taker lots)")
	flag.IntVar(&cfg.orders, "orders", 20000, "number of taker orders to blast after seeding")
	flag.Int64Var(&cfg.basePrice, "base-price", 5000, "base ask price (4-decimal scaled int)")
	flag.Int64Var(&cfg.priceStep, "price-step", 1, "step between seeded ask levels")
	flag.IntVar(&cfg.concurrency, "concurrency", 8, "number of producer goroutines blasting in parallel")
	flag.DurationVar(&cfg.drainTimeout, "drain-timeout", 90*time.Second, "max wait after blast for trade consumer to drain")
	flag.StringVar(&cfg.label, "label", "", "optional label included in stats output")
	flag.BoolVar(&cfg.noMatch, "no-match", false, "taker orders rest on the opposite side without crossing (isolates placement path, skips trade FK issue)")
	flag.StringVar(&cfg.pgDSN, "pg-dsn", "", "postgres DSN for pre-inserting taker order rows (e.g. postgres://user:pw@postgres:5432/funnyoption?sslmode=disable)")
	flag.Parse()

	cfg.brokers = strings.Split(brokersCSV, ",")
	cfg.consumerGroup = fmt.Sprintf("kafka-bench-%d-%d", time.Now().UnixNano(), os.Getpid())
	return cfg
}

type takerRecord struct {
	sentAtNanos int64
}

type stats struct {
	trades      atomic.Int64
	firstMillis atomic.Int64
	lastMillis  atomic.Int64
	matchedQty  atomic.Int64

	sentAt     []int64 // indexed by taker seq, stores UnixNano send time (atomic writes)
	latMu      sync.Mutex
	latencies  []time.Duration
	unmatched  int
	unmatchedM sync.Mutex
}

func (s *stats) observeLatency(d time.Duration) {
	s.latMu.Lock()
	s.latencies = append(s.latencies, d)
	s.latMu.Unlock()
}

func (s *stats) countUnmatched() {
	s.unmatchedM.Lock()
	s.unmatched++
	s.unmatchedM.Unlock()
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func encodeTakerID(seq int) string {
	return fmt.Sprintf("bench-t-%d", seq)
}

func decodeTakerSeq(id string) (seq int, ok bool) {
	if !strings.HasPrefix(id, "bench-t-") {
		return 0, false
	}
	n, err := strconv.Atoi(id[len("bench-t-"):])
	if err != nil {
		return 0, false
	}
	return n, true
}

func newWriter(brokers []string, topic string) *kafkago.Writer {
	return &kafkago.Writer{
		Addr:                   kafkago.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafkago.Hash{},
		BatchSize:              1000,
		BatchTimeout:           2 * time.Millisecond,
		RequiredAcks:           kafkago.RequireOne,
		Async:                  false,
		AllowAutoTopicCreation: true,
	}
}

func produceCommand(ctx context.Context, w *kafkago.Writer, cmd sharedkafka.OrderCommand) error {
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(cmd.OrderID),
		Value: payload,
	})
}

func buildMakerCmd(cfg config, level int, seq int64) sharedkafka.OrderCommand {
	price := cfg.basePrice + int64(level)*cfg.priceStep
	return sharedkafka.OrderCommand{
		CommandID:         fmt.Sprintf("bench-mk-cmd-%d-%d", level, seq),
		OrderID:           fmt.Sprintf("bench-mk-%d-%d", level, seq),
		UserID:            cfg.makerUser,
		MarketID:          cfg.market,
		Outcome:           cfg.outcome,
		Side:              "SELL",
		Type:              "LIMIT",
		TimeInForce:       "GTC",
		Price:             price,
		Quantity:          cfg.seedQty,
		CollateralAsset:   "USDT",
		RequestedAtMillis: time.Now().UnixMilli(),
	}
}

func buildTakerCmd(cfg config, seq int) sharedkafka.OrderCommand {
	takerPrice := cfg.basePrice + int64(cfg.seedLevels-1)*cfg.priceStep
	tif := "IOC"
	if cfg.noMatch {
		takerPrice = cfg.basePrice - int64(cfg.seedLevels+1)*cfg.priceStep
		if takerPrice < 1 {
			takerPrice = 1
		}
		tif = "GTC"
	}
	return sharedkafka.OrderCommand{
		CommandID:         fmt.Sprintf("bench-tk-cmd-%d", seq),
		OrderID:           encodeTakerID(seq),
		UserID:            cfg.takerUser,
		MarketID:          cfg.market,
		Outcome:           cfg.outcome,
		Side:              "BUY",
		Type:              "LIMIT",
		TimeInForce:       tif,
		Price:             takerPrice,
		Quantity:          1,
		CollateralAsset:   "USDT",
		RequestedAtMillis: time.Now().UnixMilli(),
	}
}

func preInsertTakers(ctx context.Context, cfg config) error {
	db, err := sql.Open("postgres", cfg.pgDSN)
	if err != nil {
		return fmt.Errorf("pg connect: %w", err)
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO markets (market_id, title, description, collateral_asset, status,
			open_at, close_at, resolve_at, resolved_outcome, created_by, metadata, created_at, updated_at)
		VALUES ($1, $2, '', 'USDT', 'OPEN', 0, 0, 0, '', 0, '{}'::jsonb,
			EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (market_id) DO NOTHING
	`, cfg.market, fmt.Sprintf("Market %d", cfg.market))
	if err != nil {
		return fmt.Errorf("ensure market: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO orders (order_id, client_order_id, command_id, user_id, market_id,
			outcome, side, order_type, time_in_force, collateral_asset,
			freeze_id, freeze_asset, freeze_amount,
			price, quantity, filled_quantity, remaining_quantity, status,
			cancel_reason, created_at, updated_at)
		VALUES ($1, '', '', $2, $3, $4, 'BUY', 'LIMIT', 'IOC', 'USDT',
			'', 'USDT', 0, $5, 1, 0, 1, 'NEW', '',
			EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (order_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	takerPrice := cfg.basePrice + int64(cfg.seedLevels-1)*cfg.priceStep
	for i := 0; i < cfg.orders; i++ {
		if _, err := stmt.ExecContext(ctx, encodeTakerID(i), cfg.takerUser, cfg.market, cfg.outcome, takerPrice); err != nil {
			return fmt.Errorf("insert seq %d: %w", i, err)
		}
	}
	return tx.Commit()
}

func runConsumer(ctx context.Context, cfg config, st *stats, targetTrades int, done chan<- struct{}) {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        cfg.brokers,
		GroupID:        cfg.consumerGroup,
		Topic:          cfg.prefix + "trade.matched",
		MinBytes:       1,
		MaxBytes:       10e6,
		StartOffset:    kafkago.LastOffset,
		CommitInterval: time.Second,
	})
	defer reader.Close()

	log.Printf("consumer: subscribed to %s group=%s", cfg.prefix+"trade.matched", cfg.consumerGroup)

	for {
		if ctx.Err() != nil {
			return
		}
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
				return
			}
			log.Printf("consumer: read error: %v", err)
			continue
		}
		var ev sharedkafka.TradeMatchedEvent
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		if seq, ok := decodeTakerSeq(ev.TakerOrderID); ok && seq >= 0 && seq < len(st.sentAt) {
			sentNs := atomic.LoadInt64(&st.sentAt[seq])
			if sentNs > 0 {
				latency := time.Since(time.Unix(0, sentNs))
				st.observeLatency(latency)
			}
			nowCount := st.trades.Add(1)
			st.matchedQty.Add(ev.Quantity)
			if nowCount == 1 {
				st.firstMillis.Store(time.Now().UnixMilli())
			}
			st.lastMillis.Store(time.Now().UnixMilli())
			if int(nowCount) >= targetTrades {
				select {
				case done <- struct{}{}:
				default:
				}
				return
			}
		}
	}
}

func main() {
	cfg := parseFlags()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("kafka-bench starting label=%q brokers=%v prefix=%s market=%d orders=%d seed-levels=%d seed-qty=%d concurrency=%d",
		cfg.label, cfg.brokers, cfg.prefix, cfg.market, cfg.orders, cfg.seedLevels, cfg.seedQty, cfg.concurrency)

	cmdTopic := cfg.prefix + "order.command"
	writer := newWriter(cfg.brokers, cmdTopic)
	defer writer.Close()

	log.Printf("seeding %d maker asks at prices %d..%d qty=%d user=%d",
		cfg.seedLevels, cfg.basePrice, cfg.basePrice+int64(cfg.seedLevels-1)*cfg.priceStep, cfg.seedQty, cfg.makerUser)
	seedStart := time.Now()
	for i := 0; i < cfg.seedLevels; i++ {
		cmd := buildMakerCmd(cfg, i, int64(i))
		if err := produceCommand(ctx, writer, cmd); err != nil {
			log.Fatalf("seed write failed: %v", err)
		}
	}
	log.Printf("seed phase complete in %s, waiting 2s for maker orders to rest on the book", time.Since(seedStart))
	time.Sleep(2 * time.Second)

	if cfg.pgDSN != "" {
		log.Printf("pre-inserting %d taker order rows into postgres", cfg.orders)
		if err := preInsertTakers(ctx, cfg); err != nil {
			log.Fatalf("pre-insert takers failed: %v", err)
		}
		log.Printf("pre-insert complete")
	}

	st := &stats{}
	st.sentAt = make([]int64, cfg.orders)
	st.latencies = make([]time.Duration, 0, cfg.orders)

	doneCh := make(chan struct{}, 1)
	consumerCtx, consumerCancel := context.WithCancel(ctx)
	defer consumerCancel()

	go runConsumer(consumerCtx, cfg, st, cfg.orders, doneCh)

	time.Sleep(500 * time.Millisecond) // let consumer settle on offsets

	log.Printf("blasting %d taker IOC BUY orders across %d goroutines", cfg.orders, cfg.concurrency)
	blastStart := time.Now()
	perWorker := cfg.orders / cfg.concurrency
	var wg sync.WaitGroup
	var sentTotal atomic.Int64
	for w := 0; w < cfg.concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localWriter := newWriter(cfg.brokers, cmdTopic)
			defer localWriter.Close()
			start := workerID * perWorker
			end := start + perWorker
			if workerID == cfg.concurrency-1 {
				end = cfg.orders
			}
			for i := start; i < end; i++ {
				atomic.StoreInt64(&st.sentAt[i], time.Now().UnixNano())
				cmd := buildTakerCmd(cfg, i)
				if err := produceCommand(ctx, localWriter, cmd); err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("worker %d: produce failed at seq=%d: %v", workerID, i, err)
					continue
				}
				sentTotal.Add(1)
			}
		}(w)
	}
	wg.Wait()
	blastElapsed := time.Since(blastStart)
	log.Printf("blast done: sent=%d elapsed=%s send-throughput=%.0f ops/s",
		sentTotal.Load(), blastElapsed, float64(sentTotal.Load())/blastElapsed.Seconds())

	log.Printf("draining trades (target=%d, timeout=%s)", cfg.orders, cfg.drainTimeout)
	drainStart := time.Now()
	drainTimer := time.NewTimer(cfg.drainTimeout)
	defer drainTimer.Stop()

	select {
	case <-doneCh:
		log.Printf("drain complete in %s", time.Since(drainStart))
	case <-drainTimer.C:
		log.Printf("drain timeout after %s (consumer may have missed some events)", cfg.drainTimeout)
	case <-ctx.Done():
		log.Printf("interrupted during drain")
	}
	consumerCancel()

	tradesSeen := st.trades.Load()
	matchedQty := st.matchedQty.Load()
	firstMs := st.firstMillis.Load()
	lastMs := st.lastMillis.Load()

	fmt.Println()
	fmt.Println("=======================================================")
	fmt.Println("  kafka-bench — results")
	fmt.Println("=======================================================")
	if cfg.label != "" {
		fmt.Printf("label:              %s\n", cfg.label)
	}
	fmt.Printf("brokers:            %v\n", cfg.brokers)
	fmt.Printf("prefix:             %s\n", cfg.prefix)
	fmt.Printf("market/outcome:     %d/%s\n", cfg.market, cfg.outcome)
	fmt.Printf("orders sent:        %d\n", sentTotal.Load())
	fmt.Printf("trades observed:    %d\n", tradesSeen)
	fmt.Printf("matched qty:        %d\n", matchedQty)
	fmt.Printf("send wall-clock:    %s\n", blastElapsed)
	fmt.Printf("send throughput:    %.0f orders/s\n", float64(sentTotal.Load())/blastElapsed.Seconds())

	if tradesSeen > 0 && firstMs > 0 && lastMs >= firstMs {
		tradeWindowMs := lastMs - firstMs
		if tradeWindowMs == 0 {
			tradeWindowMs = 1
		}
		fmt.Printf("first trade ts:     %s\n", time.UnixMilli(firstMs).Format(time.RFC3339Nano))
		fmt.Printf("last  trade ts:     %s\n", time.UnixMilli(lastMs).Format(time.RFC3339Nano))
		fmt.Printf("trade window:       %dms\n", tradeWindowMs)
		fmt.Printf("matching throughput:%.0f trades/s\n", float64(tradesSeen)*1000.0/float64(tradeWindowMs))
	}

	st.latMu.Lock()
	lat := make([]time.Duration, len(st.latencies))
	copy(lat, st.latencies)
	st.latMu.Unlock()

	if len(lat) > 0 {
		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		var sum time.Duration
		for _, d := range lat {
			sum += d
		}
		mean := sum / time.Duration(len(lat))
		fmt.Println()
		fmt.Printf("latency n:          %d\n", len(lat))
		fmt.Printf("latency mean:       %s\n", mean)
		fmt.Printf("latency p50:        %s\n", percentile(lat, 0.50))
		fmt.Printf("latency p90:        %s\n", percentile(lat, 0.90))
		fmt.Printf("latency p95:        %s\n", percentile(lat, 0.95))
		fmt.Printf("latency p99:        %s\n", percentile(lat, 0.99))
		fmt.Printf("latency p999:       %s\n", percentile(lat, 0.999))
		fmt.Printf("latency max:        %s\n", lat[len(lat)-1])
	}

	fmt.Println("=======================================================")

	if tradesSeen < int64(cfg.orders) {
		fmt.Fprintf(os.Stderr, "WARN: expected %d trades, observed %d (gap=%d)\n",
			cfg.orders, tradesSeen, int64(cfg.orders)-tradesSeen)
		os.Exit(2)
	}
}
