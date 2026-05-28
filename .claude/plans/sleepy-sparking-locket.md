# Plan: Eliminate per-match DB queries in TickerStream

## Context
`TickerStream.OnMatchLog` calls `newTickerMessage` every 1 second per product, which makes 2 DB queries (`GetTicksByProductId` for 24h and 30d aggregation). With 50 products, this is 100 DB queries/sec, exhausting the connection pool. The same MatchLog data is already being processed by `TickMaker` in memory — there's no need to go to the DB.

## Approach: In-memory 24h/30d ticker stats with incremental updates

Cache the 24h and 30d aggregate ticker stats (Open, Low, High, Volume) in `TickerStream`. Load from DB once on startup, then update incrementally on every `OnMatchLog`. Periodically re-sync from DB (every 5 minutes) to correct any drift.

### Key data structure

Add to `TickerStream`:
```go
type tickerStats struct {
    Open24h   decimal.Decimal
    Low24h    decimal.Decimal
    High24h   decimal.Decimal
    Volume24h decimal.Decimal
    Open30d   decimal.Decimal
    Low30d    decimal.Decimal
    High30d   decimal.Decimal
    Volume30d decimal.Decimal
    loaded    bool
    lastSync  time.Time
}
```

### Implementation steps

#### 1. Add cached stats to `TickerStream` (`pushing/ticker_stream.go`)

- Add `tickerStats` field to `TickerStream` struct
- On startup (`newTickerStream`), load 24h and 30d stats from DB using existing `service.GetTicksByProductId`
- Set `lastSync = time.Now()`

#### 2. Replace `newTickerMessage` DB queries with incremental update

- In `OnMatchLog`, before constructing the ticker message:
  - If `!s.tickerStats.loaded`: call a `loadTickerStats()` method that does the 2 DB queries, then sets `loaded = true`
  - Otherwise: update stats incrementally from the `MatchLog`:
    - `Volume24h += log.Size`, `Volume30d += log.Size`
    - `Low24h = min(Low24h, log.Price)`, `High24h = max(High24h, log.Price)` (same for 30d)
    - Open is left unchanged (it's the first trade price in the window)
  - If `time.Since(s.tickerStats.lastSync) > 5*time.Minute`: re-load from DB in a background goroutine

- `newTickerMessage` no longer takes `*MatchLog` and does DB queries. Instead it reads from `s.tickerStats` directly.

#### 3. Simplify the hot path

The `OnMatchLog` hot path becomes:
```go
func (s *TickerStream) OnMatchLog(log *matching.MatchLog, offset int64) {
    s.updateTickerStats(log)

    if (time.Now().Unix() - s.lastTickerTime) > intervalSec {
        ticker := s.newTickerMessage(log)  // reads from cached stats, no DB
        lastTickers.Store(log.ProductId, ticker)
        s.sub.publish(ChannelTicker.FormatWithProductId(log.ProductId), ticker)
        s.lastTickerTime = time.Now().Unix()
    } else {
        // fast path: just update trade fields on cached ticker
        ...
    }
    // candles path unchanged
}
```

### Files to modify

| File | Change |
|------|--------|
| `pushing/ticker_stream.go` | Add `tickerStats` struct, `loadTickerStats()`, `updateTickerStats()`, modify `newTickerMessage()` to use cache |

No other files need changes. The `service.GetTicksByProductId` function is still used, but only on init and periodic refresh — not on every match.

### Verification

1. `go build ./...` — must compile
2. Verify that `newTickerMessage` no longer calls `service.GetTicksByProductId`
3. Verify that `loadTickerStats` is called only on init and every 5 minutes
4. The ticker message output format stays identical — this is purely an internal optimization