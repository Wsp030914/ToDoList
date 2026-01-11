package initialize

import (
	"ToDoList/server/async"
	"ToDoList/server/async/handlers"
	"time"
)

/*
DeleteCOS
AvatarUpdated
*/

func InitAsyncHandlers(d *async.Dispatcher) {
	d.Register("DeleteCOS", handlers.DeleteCosObject,
		async.TimeoutPolicy{
			JobTimeout:     20 * time.Second,
			AttemptTimeout: 5 * time.Second,
			MaxRetry:       3,
		})
	d.Register("UpdateAvatar", handlers.UpdateAvatarKey,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 1 * time.Second,
			MaxRetry:       2,
		})

	d.Register("PutVersion", handlers.PutVersion,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 500 * time.Millisecond,
			MaxRetry:       2,
		})
	d.Register("PutAvatar", handlers.UpdateAvatarKey,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 1 * time.Second,
			MaxRetry:       2,
		})
	d.Register("PutProjectsSummaryCache", handlers.PutProjectsSummary,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 1 * time.Second,
			MaxRetry:       2,
		})

}
