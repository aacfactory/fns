package development

import "github.com/aacfactory/fns/clusters"

func init() {
	clusters.RegisterClusterBuilder(clusterBuilderName, clusterBuilder)
}
