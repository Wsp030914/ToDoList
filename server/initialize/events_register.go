package initialize

import (
	"ToDoList/server/async"
	"ToDoList/server/async/handlers"
	"time"
)



func InitAsyncHandlers(d *async.Dispatcher) {
	d.Register("DeleteCOS", handlers.DeleteCosObject,
		async.TimeoutPolicy{
			JobTimeout:     25 * time.Second,
			AttemptTimeout: 5 * time.Second,

		})
	d.Register("UpdateAvatar", handlers.UpdateAvatarKey,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 1 * time.Second,
		})

	d.Register("PutVersion", handlers.PutVersion,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 1 * time.Millisecond,
		})
	d.Register("PutAvatar", handlers.UpdateAvatarKey,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 1 * time.Second,
		})
	d.Register("PutProjectsSummaryCache", handlers.PutProjectsSummary,
		async.TimeoutPolicy{
			JobTimeout:     5 * time.Second,
			AttemptTimeout: 1 * time.Second,
		})

}
