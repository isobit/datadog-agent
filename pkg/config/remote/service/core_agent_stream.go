package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/DataDog/datadog-agent/pkg/api/security"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/config/remote/service/tuf"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type coreAgentStream struct {
	stream                       pbgo.AgentSecure_GetConfigUpdatesClient
	streamCtx                    context.Context
	cancel                       context.CancelFunc
	streamCancel                 context.CancelFunc
	agentClient                  pbgo.AgentSecureClient
	mx                           sync.Mutex
	currentConfigSnapshotVersion uint64
	tufClient                    *tuf.DirectorPartialClient
}

func newCoreAgentStream() (*coreAgentStream, error) {
	ctx, cancel := context.WithCancel(context.Background())

	token, err := security.FetchAuthToken()
	if err != nil {
		cancel()
		err = fmt.Errorf("unable to fetch authentication token: %w", err)
		log.Infof("unable to establish stream, will possibly retry: %s", err)
		return nil, err
	}

	streamCtx, streamCancel := context.WithCancel(
		metadata.NewOutgoingContext(ctx, metadata.MD{
			"authorization": []string{fmt.Sprintf("Bearer %s", token)},
		}),
	)

	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf(":%v", config.Datadog.GetInt("cmd_port")),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		streamCancel()
		cancel()
		return nil, err
	}

	cas := &coreAgentStream{
		streamCtx:    streamCtx,
		streamCancel: streamCancel,
		cancel:       cancel,
		agentClient:  pbgo.NewAgentSecureClient(conn),
		tufClient:    tuf.NewDirectorPartialClient(),
	}

	cas.connect()
	return cas, nil
}

func (cas *coreAgentStream) connect() {
	cas.mx.Lock()
	defer cas.mx.Unlock()
	stream := cas.getStream()
	cas.stream = stream
}

func (cas *coreAgentStream) getStream() pbgo.AgentSecure_GetConfigUpdatesClient {
	var stream pbgo.AgentSecure_GetConfigUpdatesClient
	var err error
	for {
		stream, err = cas.agentClient.GetConfigUpdates(cas.streamCtx)
		if err != nil {
			log.Errorf("Failed to establish channel to core-agent, retrying in %s...", errorRetryInterval)
			time.Sleep(errorRetryInterval)
			continue
		} else {
			log.Debugf("Successfully established channel to core-agent")
			break
		}
	}
	return stream
}

func (cas *coreAgentStream) sendTracerInfos(tracerInfos chan *pbgo.TracerInfo, product pbgo.Product) {
	for {
		select {
		case tracerInfo := <-tracerInfos:
			request := pbgo.SubscribeConfigRequest{
				CurrentConfigSnapshotVersion: cas.currentConfigSnapshotVersion,
				Product:                      product,
				TracerInfo:                   tracerInfo,
			}
			log.Trace("Sending subscribe config requests with tracer infos to core-agent")
			if err := cas.stream.Send(&request); err != nil {
				log.Warnf("Error writing tracer infos to stream: %s", err)
				time.Sleep(errorRetryInterval)
				cas.connect()
				continue
			}
		case <-cas.streamCtx.Done():
			return
		}
	}
}

func (cas *coreAgentStream) readConfigs(product pbgo.Product, callback SubscriberCallback) {
	for {
		log.Debug("Waiting for new config")
		configResponse, err := cas.stream.Recv()
		if err == io.EOF {
			continue
		} else if err != nil {
			log.Warnf("Stopped listening for configuration from remote config management: %s", err)
			time.Sleep(errorRetryInterval)
			cas.connect()
			continue
		}

		if err := cas.tufClient.Verify(configResponse); err != nil {
			log.Errorf("Partial verify failed: %s", err)
			continue
		}

		log.Infof("Got config for product %s", product)
		if err := callback(configResponse); err == nil {
			cas.currentConfigSnapshotVersion = configResponse.ConfigSnapshotVersion
		}
	}
}
