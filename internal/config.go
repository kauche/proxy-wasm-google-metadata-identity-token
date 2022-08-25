package internal

const (
	configKeyAudience                               = "audience"
	configKeyMetadataServerCluster                  = "metadata_server_cluster"
	configKeyTokenCacheDuration                     = "token_cache_duration"
	configKeyOriginalAuthorizationPropagationHeader = "original_authorization_propagation_header"
)

type pluginConfiguration struct {
	audience                               string
	metadataServerCluster                  string
	tokenCacheDuration                     uint64
	originalAuthorizationPropagationHeader string
}
