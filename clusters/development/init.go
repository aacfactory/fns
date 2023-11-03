package development

import "github.com/aacfactory/fns/clusters"

func init() {
	clusters.RegisterCluster(clusterName, new(Cluster))
}
