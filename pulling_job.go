package dlrootfs

import (
	"io"

	"github.com/docker/docker/registry"
)

type PullingJob struct {
	Session  *HubSession
	RepoData *registry.RepositoryData

	LayerId string

	LayerData io.ReadCloser
	LayerInfo []byte
	LayerSize int

	Err error
}

func NewPullingJob(session *HubSession, repoData *registry.RepositoryData, layerId string) *PullingJob {
	return &PullingJob{Session: session, RepoData: repoData, LayerId: layerId}
}

func (job *PullingJob) Start() {
	_print("\tPulling fs layer %v\n", job.LayerId)
	endpoints := job.RepoData.Endpoints
	tokens := job.RepoData.Tokens

	for _, ep := range endpoints {
		job.LayerInfo, job.LayerSize, job.Err = job.Session.GetRemoteImageJSON(job.LayerId, ep, tokens)
		if job.Err != nil {
			continue
		}
		job.LayerData, job.Err = job.Session.GetRemoteImageLayer(job.LayerId, ep, tokens, int64(job.LayerSize))
	}

	_print("\tDone %v\n", job.LayerId)
}

func (job *PullingJob) Error() error {
	return job.Err
}

func (job *PullingJob) ID() string {
	return job.LayerId
}
