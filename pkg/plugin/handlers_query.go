package plugin

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data/framestruct"
	"github.com/grafana/sentry-datasource/pkg/sentry"
)

type SentryQuery struct {
	QueryType    string   `json:"queryType"`
	OrgSlug      string   `json:"orgSlug,omitempty"`
	ProjectIds   []string `json:"projectIds,omitempty"`
	Environments []string `json:"environments,omitempty"`
	IssuesQuery  string   `json:"issuesQuery,omitempty"`
	IssuesSort   string   `json:"issuesSort,omitempty"`
	IssuesLimit  int64    `json:"issuesLimit,omitempty"`
}

func GetQuery(query backend.DataQuery) (SentryQuery, error) {
	var out SentryQuery
	err := json.Unmarshal(query.JSON, &out)
	return out, err
}

func (ds *SentryDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	dsi, err := ds.getDatasourceInstance(ctx, req.PluginContext)
	if err != nil {
		response.Responses["error"] = backend.DataResponse{Error: err}
		return response, nil
	}
	for _, q := range req.Queries {
		res := QueryData(ctx, req.PluginContext, q, dsi.sentryClient)
		response.Responses[q.RefID] = res
	}
	return response, nil
}

func QueryData(ctx context.Context, pCtx backend.PluginContext, backendQuery backend.DataQuery, client sentry.SentryClient) backend.DataResponse {
	response := backend.DataResponse{}
	query, err := GetQuery(backendQuery)
	if err != nil {
		return GetErrorResponse(response, "", err)
	}
	switch query.QueryType {
	case "issues":
		if query.OrgSlug == "" {
			return GetErrorResponse(response, "", ErrorInvalidOrganizationSlug)
		}
		issues, executedQueryString, err := client.GetIssues(sentry.GetIssuesInput{
			OrganizationSlug: query.OrgSlug,
			ProjectIds:       query.ProjectIds,
			Environments:     query.Environments,
			Query:            query.IssuesQuery,
			Sort:             query.IssuesSort,
			Limit:            query.IssuesLimit,
			From:             backendQuery.TimeRange.From,
			To:               backendQuery.TimeRange.To,
		})
		if err != nil {
			return GetErrorResponse(response, executedQueryString, err)
		}
		frame, err := framestruct.ToDataFrame(GetFrameName("Issues", backendQuery.RefID), issues)
		if err != nil {
			return GetErrorResponse(response, executedQueryString, err)
		}
		frame = UpdateFrameMeta(frame, executedQueryString, query, client.BaseURL)
		response.Frames = append(response.Frames, frame)
	default:
		response.Error = ErrorUnknownQueryType
	}
	return response
}