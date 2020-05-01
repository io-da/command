package command

//
//import (
//	"sync"
//	"sync/atomic"
//	"time"
//)
//
//// ScheduledCommandsAdapter is the struct used for memory caching purposes.
//type ScheduledCommandsAdapter struct {
//	sync.Mutex
//	cachedResults map[schedule.Schedule]Command
//	cleanerSignal chan bool
//	shuttingDown  *uint32
//	sleepTimer    *time.Timer
//	sleepUntil    time.Time
//}
//
//// NewScheduledCommandsAdapter initializes a new *ScheduledCommandsAdapter.
//// This function will also initialize the respective cleaner routine.
//func NewScheduledCommandsAdapter() *ScheduledCommandsAdapter {
//	ad := &ScheduledCommandsAdapter{
//		cachedResults: make(map[string]*Result),
//		cleanerSignal: make(chan bool, 1),
//		shuttingDown:  new(uint32),
//	}
//	go ad.cleaner()
//	return ad
//}
//
//// Set stores the cache value for the given query.
//func (ad *ScheduledCommandsAdapter) Set(qry Cacheable, res *Result, at time.Time) bool {
//	ad.Lock()
//	ad.cachedResults[string(qry.CacheKey())] = res
//	ad.updateSleepUntil(at.Add(qry.CacheDuration()))
//	ad.clean()
//	ad.Unlock()
//	return true
//}
//
//// Get retrieves the cached result for the provided query.
//func (ad *ScheduledCommandsAdapter) Get(qry Cacheable) *Result {
//	ad.Lock()
//	res, isCached := ad.cachedResults[string(qry.CacheKey())]
//	ad.Unlock()
//	if isCached {
//		return res
//	}
//	return nil
//}
//
//// Expire can optionally be used to forcibly expire a query cache.
//func (ad *ScheduledCommandsAdapter) Expire(qry Cacheable) {
//	ck := string(qry.CacheKey())
//	ad.Lock()
//	if _, isCached := ad.cachedResults[ck]; isCached {
//		delete(ad.cachedResults, ck)
//	}
//	ad.Unlock()
//}
//
//// Shutdown is used to stop the cleaner routine.
//func (ad *ScheduledCommandsAdapter) Shutdown() {
//	atomic.CompareAndSwapUint32(ad.shuttingDown, 0, 1)
//	ad.clean()
//}
//
////------Internal------//
//
//func (ad *ScheduledCommandsAdapter) cleaner() {
//	for atomic.LoadUint32(ad.shuttingDown) == 0 {
//		ad.Lock()
//		now := time.Now()
//		ad.sleepUntil = time.Time{}
//		for key, res := range ad.cachedResults {
//			if !res.CachedAt().IsZero() && now.After(res.ExpiresAt()) {
//				delete(ad.cachedResults, key)
//				continue
//			}
//			ad.updateSleepUntil(res.ExpiresAt())
//		}
//		ad.updateSleepTimer(ad.determineSleepDuration())
//		ad.Unlock()
//
//		// allow the cleaner to be triggered either with timer or directly
//		select {
//		case <-ad.sleepTimer.C:
//		case <-ad.cleanerSignal:
//		}
//	}
//}
//
//func (ad *ScheduledCommandsAdapter) clean() {
//	ad.cleanerSignal <- true
//}
//
//func (ad *ScheduledCommandsAdapter) updateSleepUntil(expiresAt time.Time) {
//	if ad.sleepUntil.IsZero() || expiresAt.Before(ad.sleepUntil) {
//		ad.sleepUntil = expiresAt
//	}
//}
//
//func (ad *ScheduledCommandsAdapter) determineSleepDuration() time.Duration {
//	if ad.sleepUntil.IsZero() || len(ad.cachedResults) <= 0 {
//		return time.Hour
//	}
//
//	return ad.sleepUntil.Sub(time.Now())
//}
//
//func (ad *ScheduledCommandsAdapter) updateSleepTimer(d time.Duration) {
//	if ad.sleepTimer == nil {
//		ad.sleepTimer = time.NewTimer(d)
//		return
//	}
//	ad.sleepTimer.Reset(d)
//}
