package handlers

import (
	"ToDoList/server/async"
	"ToDoList/server/service"
	"context"
	"encoding/json"
	"go.uber.org/zap"
)

type GetProjectSummaryPayload struct {
	Items []service.ProjectSummary `json:"items"`
	Total int64                    `json:"total"`
	UID   int                      `json:"uid"`
	Ver   int64                    `json:"ver"`
	Name  string                   `json:"name"`
	Page  int                      `json:"page"`
	Size  int                      `json:"size"`
}

func PutProjectsSummary(ctx context.Context, job async.Job, lg *zap.Logger) error {
	var g GetProjectSummaryPayload
	if err := json.Unmarshal(job.Payload, &g); err != nil {
		lg.Error(job.Type + job.TraceID + "Payload Unmarshal is err")
		return nil
	}

	err := service.PutProjectsSummaryCache(ctx, g.UID, g.Name, g.Page, g.Size, g.Total, g.Items, g.Ver)
	service.PutTraceID(ctx, job.Type, job.TraceID, err)
	return err
}
