package local

import (
	"bytes"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"gopkg.in/inconshreveable/log15.v2"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sqs/pbtypes"
	authpkg "src.sourcegraph.com/sourcegraph/auth"
	"src.sourcegraph.com/sourcegraph/util/metricutil"
)

var GraphUplink sourcegraph.GraphUplinkServer = &graphUplink{}

type graphUplink struct{}

var _ sourcegraph.GraphUplinkServer = (*graphUplink)(nil)

func (s *graphUplink) Push(ctx context.Context, snapshot *sourcegraph.MetricsSnapshot) (*pbtypes.Void, error) {
	defer noCache(ctx)

	actorID := authpkg.ActorFromContext(ctx).ClientID
	log15.Debug("GraphUplink metrics push", "actorID", actorID, "type", snapshot.Type, "dataSize", len(snapshot.TelemetryData))

	if actorID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Could not find ClientID")
	}
	if snapshot.Type != sourcegraph.TelemetryType_PrometheusDelimited0dot0dot4 {
		return nil, grpc.Errorf(codes.InvalidArgument, "GraphUplink.Push only support PrometheusDelimited0dot0dot4")
	}

	mfs := metricutil.UnmarshalMetricFamilies(bytes.NewBuffer(snapshot.TelemetryData))
	log15.Debug("GraphUplink metrics push", "actorID", actorID, "numMetrics", len(mfs))

	err := mfs.AnnotateWithClientID(actorID)
	if err != nil {
		return nil, err
	}

	sanitizedActorID := strings.SplitN(actorID, "/", 2)[0]
	err = mfs.PushToGateway("push.metrics.sgdev.org", "downstream_src", sanitizedActorID)
	if err != nil {
		log15.Warn("Failed to push client metrics to gateway", "error", err)
	}

	return &pbtypes.Void{}, nil
}

func (s *graphUplink) PushEvents(ctx context.Context, eventList *sourcegraph.UserEventList) (*pbtypes.Void, error) {
	defer noCache(ctx)

	actorID := authpkg.ActorFromContext(ctx).ClientID

	if actorID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Could not find ClientID")
	}

	for _, event := range eventList.Events {
		event.ClientID = actorID
	}
	log15.Debug("GraphUplink events push", "actorID", actorID, "numEvents", len(eventList.Events))

	go metricutil.ForwardEvents(ctx, eventList)

	return &pbtypes.Void{}, nil
}
